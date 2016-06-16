// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// madvise, mincore are not implemented on ESX

package runtime

import (
	"unsafe"
	"runtime/internal/atomic"
)

type spinlock struct {
	v uint32
}

//go:nosplit
func (l *spinlock) lock() {
	for {
		if atomic.Cas(&l.v, 0, 1) {
			return
		}
		osyield()
	}
}

//go:nosplit
func (l *spinlock) unlock() {
	atomic.Store(&l.v, 0)
}

const (
	_EACCES    = 13
)

var mmapReserveFd = int32(-1)
var mmapReserveFile = []byte("/tmp/mmapfix_reserve\x00")
var mmapReserveLow uintptr
var mmapReserveHigh uintptr
var mmapLock spinlock

func esxhalt()

//go:nosplit
func throwWithStatus(s string, c uintptr) {
	print("fatal error: ", s, "code = ", c, "\n")
	gp := getg()
	if gp.m.throwing == 0 {
		gp.m.throwing = 1
	}
	esxhalt()
	startpanic()
	dopanic(0)
	*(*int)(nil) = 0 // not reached
}

// Don't split the stack as this method may be invoked without a valid G, which
// prevents us from allocating more stack.
//go:nosplit
func sysAlloc(n uintptr, sysStat *uint64) unsafe.Pointer {
	mmapLock.lock()
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
	} else if uintptr(p) >= mmapReserveLow && uintptr(p) < mmapReserveHigh {
		throwWithStatus("mmap conflict", uintptr(p))
	}
	mSysStatInc(sysStat, n)
	mmapLock.unlock()
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
	if uintptr(v) >= mmapReserveLow && uintptr(v) + n <= mmapReserveHigh {
		p := mmap(v, n, _PROT_NONE, _MAP_PRIVATE | _MAP_NORESERVE | _MAP_FIXED, mmapReserveFd, uint32(uintptr(v) - mmapReserveLow))
		if p!= v {
			throwWithStatus("failed to sysFree a reserved area", uintptr(p))
		}
	} else {
		munmap(v, n)
	}
}

func sysFault(v unsafe.Pointer, n uintptr) {
	mmap(v, n, _PROT_NONE, _MAP_ANON|_MAP_PRIVATE|_MAP_FIXED, -1, 0)
}

func sysReserve(v unsafe.Pointer, n uintptr, reserved *bool) unsafe.Pointer {
	// FIXME: should be protected by lock
	if mmapReserveFd == -1 {
		mmapReserveFd = open(&mmapReserveFile[0], _O_RDONLY | _O_CLOEXEC | _O_CREAT, 0644)
	}
	if mmapReserveLow != mmapReserveHigh {
		throwWithStatus("cannot reserve twice", 0)
	}
	p := mmap(v, n, _PROT_NONE, _MAP_PRIVATE | _MAP_NORESERVE, mmapReserveFd, 0)
	if p != v {
		if uintptr(p) >= 4096 {
			munmap(p, n)
		}
		return nil
	}
	mmapReserveLow = uintptr(v)
	mmapReserveHigh = uintptr(v) + n
	*reserved = true
	return p
}

func sysMap(v unsafe.Pointer, n uintptr, reserved bool, sysStat *uint64) {
	mmapLock.lock()
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
		mmapLock.unlock()
		return
	}

	p := mmap(v, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_FIXED|_MAP_PRIVATE, -1, 0)
	if uintptr(p) == _ENOMEM {
		throw("runtime: out of memory")
	}
	if p != v {
		throwWithStatus("runtime: cannot map pages in arena address space", uintptr(p))
	}
	mmapLock.unlock()
}