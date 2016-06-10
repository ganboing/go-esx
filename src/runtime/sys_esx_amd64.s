// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
// System calls and other sys.stuff for AMD64, ESX
//

#include "go_asm.h"
#include "go_tls.h"
#include "textflag.h"

// func now() (sec int64, nsec int32)
TEXT time·now(SB),NOSPLIT,$16
	LEAQ	0(SP), DI
	MOVQ	$0, SI
	MOVL	$96, AX
	SYSCALL
	MOVQ	0(SP), AX	// sec
	MOVL	8(SP), DX	// usec
	IMULQ	$1000, DX
	MOVQ	AX, sec+0(FP)
	MOVL	DX, nsec+8(FP)
	RET

TEXT runtime·nanotime(SB),NOSPLIT,$16
	LEAQ	0(SP), DI
	MOVQ	$0, SI
	MOVL	$96, AX
	SYSCALL
	MOVQ	0(SP), AX	// sec
	MOVL	8(SP), DX	// usec
	IMULQ	$1000, DX
	// sec is in AX, nsec in DX
	// return nsec in AX
	IMULQ	$1000000000, AX
	ADDQ	DX, AX
	MOVQ	AX, ret+0(FP)
	RET

// Likewise
// $16 to account for potentially not 16 byte aligned SP
TEXT runtime·callCgoMmapReserve(SB),NOSPLIT,$16
	MOVQ	addr+0(FP), DI
	MOVQ	n+8(FP), SI
	MOVQ	_cgo_mmap_reserve(SB), AX
	MOVQ	SP, BX
	ANDQ	$~15, SP
	MOVQ	BX, 0(SP)
	CALL	AX
	MOVQ	0(SP), SP
	MOVQ	AX, ret+16(FP)
	RET

