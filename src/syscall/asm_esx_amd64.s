// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"
#include "funcdata.h"

//
// System calls for AMD64, ESX
//

// C api: VmkuserStatus_Code VmkuserCompat_ForkExec(const char* filePath, char* const* argv, char* const* envp, int32 workingDirFd, int32* initFds, uint32 numInitFds, int32 uid, int32 gid, Bool detached, uint32* outPid);
// go api: func VMKForkExec(filepath *byte, argv, envp []*byte, wdfd int32, initfds []int32, initfdslength uint32, uid, gid int32, detached bool, pid *uint32) uintptr
TEXT ·VMKForkExec(SB),NOSPLIT,$24
	MOVQ	filepath+0(FP), DI
	MOVQ	argv+8(FP), SI
	MOVQ	envp+16(FP), DX
	MOVL	workingDirFd+24(FP), R10
	MOVQ	initFds+32(FP), R8
	MOVL	initFdLength+40(FP), AX
	MOVL	AX, 0(SP)
	MOVL	uid+44(FP), AX
	MOVL	AX, 4(SP)
	MOVL	gid+48(FP), AX
	MOVL	AX, 8(SP)
	MOVL	detached+52(FP), AX
	MOVB	AX, 12(SP)
	MOVQ	pid+56(FP), AX
	MOVQ	AX, 13(SP)
	MOVQ	$1025, AX
	MOVQ	SP, R9
	SYSCALL
	MOVL	DX, ret+64(FP)
// VMKernel use AX for VMK status and DX for Linux status
	RET
