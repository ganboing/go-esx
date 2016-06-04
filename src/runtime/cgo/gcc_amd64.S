// Copyright 2009 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Apple still insists on underscore prefixes for C function names.
 */
#if defined(__APPLE__)
#define EXT(s) _##s
#else
#define EXT(s) s
#endif

/*
 * void crosscall_amd64(void (*fn)(void))
 *
 * Calling into the 6c tool chain, where all registers are caller save.
 * Called from standard x86-64 ABI, where %rbx, %rbp, %r12-%r15
 * are callee-save so they must be saved explicitly.
 * The standard x86-64 ABI passes the three arguments m, g, fn
 * in %rdi, %rsi, %rdx.
 */
.globl EXT(crosscall_amd64)
EXT(crosscall_amd64):
	pushq %rbx
	pushq %rbp
	pushq %r12
	pushq %r13
	pushq %r14
	pushq %r15

#if defined(_WIN64)
	call *%rcx	/* fn */
#else
	call *%rdi	/* fn */
#endif

	popq %r15
	popq %r14
	popq %r13
	popq %r12
	popq %rbp
	popq %rbx
	ret

.extern do_mmap_redir
.extern do_mremap_redir
.extern do_munmap_redir

.globl mmap_redir
mmap_redir:
	movq %r10, %rcx
	call do_mmap_redir
	cmp $-0xfff, %rax
	ret

.globl mremap_redir
mremap_redir:
	movq %r10, %rcx
	call do_mremap_redir
	cmp $-0xfff, %rax
	ret

.globl munmap_redir
munmap_redir:
	call do_munmap_redir
	cmp $-0xfff, %rax
	ret

raw_syscall:
	movq %rcx, %r10
	syscall
	ret

.globl raw_sys_mmap
raw_sys_mmap:
	movl $9, %eax
	jmp raw_syscall

.globl raw_sys_munmap
raw_sys_munmap:
	movl $11, %eax
	jmp raw_syscall

.globl raw_sys_mremap
raw_sys_mremap:
	movl $25, %eax
	jmp raw_syscall

.globl raw_sys_sched_yield
raw_sys_sched_yield:
	movl $24, %eax
	jmp raw_syscall

#ifdef __ELF__
.section .note.GNU-stack,"",@progbits
#endif
