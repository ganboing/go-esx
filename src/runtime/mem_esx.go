// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// madvise, mincore are not implemented on ESX

package runtime

import (
	"unsafe"
)

const (
	_EACCES    = 13
)

var mmapReserveFd = int32(-1)
var mmapReserveFile = []byte("/tmp/mmapfix_reserve\x00")

// Don't split the stack as this method may be invoked without a valid G, which
// prevents us from allocating more stack.
//go:nosplit
func sysAlloc(n uintptr, sysStat *uint64) unsafe.Pointer {
	p := mmap(nil, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_PRIVATE, -1, 0)
	if uintptr(p) < 4096 {
		if uintptr(p) == _EACCES {
			print("runtime: mmap: access denied\n")
			exit(2)
		}
		if uintptr(p) == _EAGAIN {
			print("runtime: mmap: too much locked memory (check 'ulimit -l').\n")
			exit(2)
		}
		return nil
	}
	mSysStatInc(sysStat, n)
	return p
}

func sysUnused(v unsafe.Pointer, n uintptr) {
}

func sysUsed(v unsafe.Pointer, n uintptr) {
}

// Don't split the stack as this function may be invoked without a valid G,
// which prevents us from allocating more stack.
//go:nosplit
func sysFree(v unsafe.Pointer, n uintptr, sysStat *uint64) {
	mSysStatDec(sysStat, n)
	munmap(v, n)
}

func sysFault(v unsafe.Pointer, n uintptr) {
	mmap(v, n, _PROT_NONE, _MAP_ANON|_MAP_PRIVATE|_MAP_FIXED, -1, 0)
}

func sysReserve(v unsafe.Pointer, n uintptr, reserved *bool) unsafe.Pointer {
	if mmapReserveFd == -1 {
		// FIXME: should be protected by lock
		mmapReserveFd = open(&mmapReserveFile[0], _O_RDONLY | _O_CLOEXEC | _O_CREAT, 0644)
	}
	p := mmap(v, n, _PROT_NONE, _MAP_PRIVATE | _MAP_NORESERVE, mmapReserveFd, 0)
	if p != v {
		return nil
	}
	*reserved = true
	return p
}

func sysMap(v unsafe.Pointer, n uintptr, reserved bool, sysStat *uint64) {
	mSysStatInc(sysStat, n)

	if !reserved {
		p := mmap(v, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_PRIVATE, -1, 0)
		if uintptr(p) == _ENOMEM {
			throw("runtime: out of memory")
		}
		if p != v {
			print("runtime: address space conflict: map(", v, ") = ", p, "\n")
			throw("runtime: address space conflict")
		}
		return
	}

	p := mmap(v, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_FIXED|_MAP_PRIVATE, -1, 0)
	if uintptr(p) == _ENOMEM {
		throw("runtime: out of memory")
	}
	if p != v {
		throw("runtime: cannot map pages in arena address space")
	}
}