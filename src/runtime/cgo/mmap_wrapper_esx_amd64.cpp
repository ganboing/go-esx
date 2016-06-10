#include <dlfcn.h>
#include <unistd.h>
#include <sys/mman.h>
#include <sys/syscall.h>
#include <link.h>
#include <cstdio>
#include <cstdint>
#include <cstring>
#include <atomic>
#include <mutex>
#include <malloc.h>
#include <sched.h>
#include <errno.h>
#include "mmap_wrapper_esx_amd64.h"

// +build cgo

// +build esx,amd64

template<size_t n>
void debug_output(const char (&str)[n])
{
	write(STDERR_FILENO, str, n-1);
}

#ifndef PAGE_SIZE
#define PAGE_SIZE (4096UL)
#endif

//void crash() __attribute__((noreturn));
template<size_t n>
inline void crash(const char (&str)[n]) {
	debug_output(str);
	*(char*)NULL = 0;
	__builtin_unreachable();
}

struct func_patch_info {
	unsigned int& lib_offset;
	unsigned int syscall_no;
	void* redir;
	const char* name;
};

extern "C" long mmap_redir(void *addr, size_t length, int prot, int flags, int fd, off_t offset);
extern "C" long raw_sys_mmap(void *addr, size_t length, int prot, int flags, int fd, off_t offset);
extern "C" long munmap_redir(void* addr, size_t len);
extern "C" long raw_sys_munmap(void* addr, size_t len);
extern "C" long mremap_redir(void* old_address, size_t old_size, size_t new_size, int flags, void* new_address);
extern "C" long raw_sys_mremap(void* old_address, size_t old_size, size_t new_size, int flags, void* new_address);

//for ld
const static func_patch_info ld_patches[]=
{
	{mmap_ld_patch_offset, 9, (void*)mmap_redir, "mmap"},
	{munmap_ld_patch_offset, 11, (void*)munmap_redir, "munmap"}
};

//for libc
const static func_patch_info libc_patches[]=
{
	{mmap_libc_patch_offset, 9, (void*)mmap_redir, "mmap"},
	{mremap_libc_patch_offset, 25, (void*)mremap_redir, "mremap"},
	{munmap_libc_patch_offset, 11, (void*)munmap_redir, "munmap"}
};

struct jmp{
	char op;
	unsigned int offset;
}__attribute__ ((packed));

#define SYSCALL_SEQ1 "\xB8"
#define SYSCALL_SEQ2 "\x0F\x05\x48\x3D\x01\xF0\xFF\xFF"
#define PATCH_SEQ1 "\x48\xB8"
#define PATCH_SEQ2 "\xFF\xD0\x90"

struct syscall_prolog{
	char seq1[sizeof(SYSCALL_SEQ1)-1];
	int no;
	char seq2[sizeof(SYSCALL_SEQ2)-1];
}__attribute__ ((packed));

struct syscall_patch{
	char _seq1[sizeof(PATCH_SEQ1)-1];
	long addr;
	char _seq2[sizeof(PATCH_SEQ2)-1];
}__attribute__ ((packed));

static_assert(sizeof(syscall_prolog) == sizeof(syscall_patch), "check patch point size");

static void* hook(void* point, void* redir, int n){
	syscall_prolog orig;
	syscall_patch patch;
	memcpy(orig.seq1, SYSCALL_SEQ1, sizeof(orig.seq1));
	orig.no = n;
	memcpy(orig.seq2, SYSCALL_SEQ2, sizeof(orig.seq2));
	if(memcmp(&orig, point, sizeof(orig)))
		return NULL;

	uintptr_t align = (uintptr_t)point & -PAGE_SIZE;
	size_t prot_size = (uintptr_t)point - align + sizeof(patch);

	mprotect((void*)align, prot_size, PROT_WRITE | PROT_EXEC);
	memcpy(patch._seq1, PATCH_SEQ1, sizeof(patch._seq1));
	patch.addr = (long)redir;
	memcpy(patch._seq2, PATCH_SEQ2, sizeof(patch._seq2));	
	memcpy(point, &patch, sizeof(patch));
	mprotect((void*)align, prot_size, PROT_EXEC);
	return point;
}

template<const char *lib>
struct patch_lib{
	template<size_t n>
	inline patch_lib(const func_patch_info (&hooks)[n]) {
		link_map* map = (link_map*)dlopen(lib, RTLD_LAZY | RTLD_GLOBAL | RTLD_NOLOAD);
		if(!map)
			crash("unable to find link_map");
		for(size_t i=0; i!=n; ++i){
			void* addr;
			if((addr = hook((void*)(map->l_addr + hooks[i].lib_offset), (void*)hooks[i].redir, hooks[i].syscall_no)))
				fprintf(stderr, "patched %s!%s @ %p\n", lib, hooks[i].name, addr);
		}
	}
};

extern const char LD_PATH[] = "ld-linux-x86-64.so.2";
extern const char LIBC_PATH[] = "libc.so.6";

extern "C" long raw_sys_sched_yield();

struct spinlock : std::atomic_bool
{
	inline void lock(){
		bool prev = false;
		while(!this->compare_exchange_weak(prev, true)){
			prev = false;
			raw_sys_sched_yield();
		}
	}
	inline void unlock(){
		this->store(false);
	}
};

static spinlock  mm_lock;

static bool syscall_failed(unsigned long status)
{
	return status > (unsigned long)-0x1000LL;
}

static uintptr_t mmap_reserved_low = 0;
static uintptr_t mmap_reserved_high = 0;

typedef struct malloc_chunk *mbinptr;
#define NBINS             128
struct malloc_save_state
{ 
  long magic;
  long version;
  mbinptr av[NBINS * 2 + 2];
  char *sbrk_base;
  int sbrked_mem_bytes;
  unsigned long trim_threshold;
  unsigned long top_pad;
  unsigned int n_mmaps_max;
  unsigned long mmap_threshold;
  int check_action;
  unsigned long max_sbrked_mem;
  unsigned long max_total_mem;
  unsigned int n_mmaps;
  unsigned int max_n_mmaps;
  unsigned long mmapped_mem;
  unsigned long max_mmapped_mem;
  int using_malloc_checking;
  unsigned long max_fast;
  unsigned long arena_test;
  unsigned long arena_max;
  unsigned long narenas;
};
static const malloc_save_state* malloc_state;
#undef NBINS
static const uintptr_t mmap_base = 0x100000000ULL - PAGE_SIZE;

struct _mmap_init_base{
	inline _mmap_init_base(){
		malloc_state = (const malloc_save_state*)malloc_get_state();
		fprintf(stderr, "Assuming %p~%p is mmap'ed already\n", 
			malloc_state->sbrk_base, malloc_state->sbrk_base + 0x100000000ULL);
		if(syscall_failed(raw_sys_mmap((void*)mmap_base, PAGE_SIZE, PROT_READ, 
			MAP_PRIVATE|MAP_ANONYMOUS|MAP_FIXED, 0, 0)))
			crash("failed to init base");
	}
};

#pragma GCC visibility push(default)
extern "C" void* mmapfix_reserve(void* addr, size_t length){
	static _mmap_init_base mmap_init_base;
	static patch_lib<LD_PATH> patch_ld(ld_patches);
	static patch_lib<LIBC_PATH> patch_libc(libc_patches); 
	std::lock_guard<spinlock> lock(mm_lock);
	//currently only supports one reservation
	if(mmap_reserved_low != mmap_reserved_high || (uintptr_t)addr < mmap_base)
		return NULL;
	uintptr_t mmapped_region_low = (uintptr_t)malloc_state->sbrk_base;
	uintptr_t mmapped_region_high = mmapped_region_low + 0x100000000ULL;
	if((uintptr_t)addr < mmapped_region_high && (uintptr_t)addr + length > mmapped_region_low)
		return NULL;
	mmap_reserved_low = (uintptr_t)addr;
	mmap_reserved_high = (uintptr_t)addr + length;
	fprintf(stderr, "mmapfix: reserved range %p-%p\n", (void*)mmap_reserved_low, (void*)mmap_reserved_high);
	return addr;
}
#pragma GCC visibility pop

static const unsigned long mmap_max_retry = 0x10;
uintptr_t mmap_retries[mmap_max_retry];
unsigned long mmap_retry_cnt = 0;

extern "C" long do_mmap_redir(void *addr, size_t length, int prot, int flags,
                  int fd, off_t offset)
{
	std::lock_guard<spinlock> lock(mm_lock);
	mmap_retry_cnt = 0;
	unsigned long ret; //*unsigned* is critical
	do{
		ret = raw_sys_mmap(addr, length, prot, flags, fd, offset);
		if(syscall_failed(ret))
			break;
		if(!(flags & MAP_FIXED) && ret >= mmap_reserved_low && ret < mmap_reserved_high){
			if(syscall_failed(raw_sys_munmap((void*)ret, length)) ||
				syscall_failed(raw_sys_munmap((void*)mmap_base, PAGE_SIZE)) ||
				syscall_failed(raw_sys_mmap((void*)mmap_base, PAGE_SIZE, PROT_READ, 
					MAP_PRIVATE|MAP_ANONYMOUS|MAP_FIXED, 0, 0)))
				crash("mmap/munmap failed");
			addr = (void*)mmap_base;
		}
		else
			break;
		mmap_retries[mmap_retry_cnt] = ret;
	}while(++mmap_retry_cnt < mmap_max_retry);
	if(mmap_retry_cnt == mmap_max_retry){
		__asm__ __volatile__ ("int3");
		ret = -EAGAIN;
	}
	__asm__ __volatile__ ("nop"); //gdb hook point
	return ret;
}
extern "C" long do_munmap_redir(void* addr, size_t len)
{
	std::lock_guard<spinlock> lock(mm_lock);
	return raw_sys_munmap(addr, len);
}
extern "C" long do_mremap_redir(void* old_address, size_t old_size, size_t new_size, int flags, void* new_address)
{
	std::lock_guard<spinlock> lock(mm_lock);
	return raw_sys_mremap(old_address, old_size, new_size, flags, new_address);
}
