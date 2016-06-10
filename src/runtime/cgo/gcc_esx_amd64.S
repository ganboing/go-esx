// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
