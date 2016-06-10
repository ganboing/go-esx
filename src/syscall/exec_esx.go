// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build esx

package syscall

import (
	"unsafe"
)

type SysProcAttr struct {
	Credential  *Credential    // Credential.
}

// Implemented in runtime package.
func runtime_BeforeFork()
func runtime_AfterFork()

// on ESX, ForkExec expects a array of uint32, not uintptr
func fdsIntFromUintptr(fds []uintptr) [] int32 {
	fdsInt := make([]int32, len(fds))
	for i := 0; i < len(fds); i++ {
		fdsInt[i] = int32(fds[i])
	}
	return fdsInt
}

func forkExec(argv0 string, argv []string, attr *ProcAttr) (pid int, err error) {
	var err1 Errno

	if attr == nil {
		attr = &zeroProcAttr
	}
	sys := attr.Sys
	if sys == nil {
		sys = &zeroSysProcAttr
	}

	// Convert args to C form.
	argv0p, err := BytePtrFromString(argv0)
	if err != nil {
		return 0, err
	}
	argvp, err := SlicePtrFromStrings(argv)
	if err != nil {
		return 0, err
	}
	envvp, err := SlicePtrFromStrings(attr.Env)
	if err != nil {
		return 0, err
	}

	var dir *byte
	if attr.Dir != "" {
		dir, err = BytePtrFromString(attr.Dir)
		if err != nil {
			return 0, err
		}
	}
	
	// Kick off child.
	pid, err1 = forkExecEsx(argv0p, argvp, envvp, dir, attr, sys)
	if err1 != 0 {
		err = Errno(err1)
		return 0, err
	}
	return pid, nil
}

func forkExecEsx(argv0 *byte, argv, envv []*byte, dir *byte, attr *ProcAttr, sys *SysProcAttr) (int, Errno) {
	var (
		r1		uintptr
		uid		int32 = -1
		gid 	int32 = -1
		wdfd	int32 = -1
		pid		uint32
		err		Errno
	)
	Fds := fdsIntFromUintptr(attr.Files)
	if sys.Credential != nil {
		uid = int32(sys.Credential.Uid)
		gid = int32(sys.Credential.Gid)
	}
	if dir != nil {
		r1, _, err = RawSyscall(SYS_OPEN, uintptr(unsafe.Pointer(dir)), O_RDONLY, 0)
		if err != 0 {
			return 0, err
		}
		wdfd = int32(r1)
	} 
	err = VMKForkExec(argv0, &argv[0], &envv[0], wdfd, &Fds[0], uint32(len(Fds)), uid, gid, false, &pid)
	RawSyscall(SYS_CLOSE, uintptr(wdfd), 0, 0)
	return int(pid), err
}
