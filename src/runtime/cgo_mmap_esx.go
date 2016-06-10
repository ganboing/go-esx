// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Support for memory sanitizer.  See runtime/cgo/mmap.go.

// +build esx,amd64

package runtime

import "unsafe"
import "runtime/internal/atomic"

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

// _cgo_mmap is filled in by runtime/cgo when it is linked into the
// program, so it is only non-nil when using cgo.
//go:linkname _cgo_mmap _cgo_mmap
var _cgo_mmap unsafe.Pointer

//go:linkname _cgo_mmap_reserve _cgo_mmap_reserve
var _cgo_mmap_reserve unsafe.Pointer

var mmap_reserve_low uintptr
var mmap_reserve_high uintptr

var mmapLock spinlock

const (
	mmap_base = uintptr(0x30000000000)
	max_retry = uint(0x10)
)

func mmapreserveinit() {
	if _cgo_mmap_reserve == nil && uintptr(sysMmap(unsafe.Pointer(mmap_base), 4096, _PROT_READ, _MAP_PRIVATE | _MAP_ANON | _MAP_FIXED, -1, 0)) != mmap_base {
		throw("mmap_reserve init failed")
	}
}

func mmap_reserve(addr unsafe.Pointer, n uintptr) unsafe.Pointer{
	if _cgo_mmap_reserve != nil {
		var ret uintptr
		systemstack(func() {
			ret = callCgoMmapReserve(addr, n)
		})
		return unsafe.Pointer(ret)
	} else if mmap_reserve_low == mmap_reserve_high && uintptr(addr) + n <= mmap_base {
		mmap_reserve_low = uintptr(addr)
		mmap_reserve_high = mmap_reserve_low + n
		return addr
	}
	return nil;
}

func mmap(addr unsafe.Pointer, n uintptr, prot, flags, fd int32, off uint32) unsafe.Pointer {
	if _cgo_mmap != nil {
		// Make ret a uintptr so that writing to it in the
		// function literal does not trigger a write barrier.
		// A write barrier here could break because of the way
		// that mmap uses the same value both as a pointer and
		// an errno value.
		// TODO: Fix mmap to return two values.
		var ret uintptr
		systemstack(func() {
			ret = callCgoMmap(addr, n, prot, flags, fd, off)
		})
		return unsafe.Pointer(ret)
	}
	var (
		cnt uint
		ret uintptr
	)
	mmapLock.lock()
	for cond := true; cond; cond = cnt < max_retry {
		ret = uintptr(sysMmap(addr, n, prot, flags, fd, off))
		if ret < 4096 {
			break
		}
		if flags & _MAP_FIXED == 0 && ret >= mmap_reserve_low && ret < mmap_reserve_high {
			munmap(unsafe.Pointer(ret), n)
			munmap(unsafe.Pointer(mmap_base), 4096)
			if uintptr(sysMmap(unsafe.Pointer(mmap_base), 4096, _PROT_READ, _MAP_PRIVATE | _MAP_ANON | _MAP_FIXED, -1, 0)) != mmap_base {
				mmapLock.unlock()
				throw("mmap base failed")
			}
			addr = unsafe.Pointer(mmap_base + 4096)
			cnt = cnt + 1
		} else {
			break;
		}
	}
	mmapLock.unlock()
	if cnt == max_retry {
		throw("too many retries")
	}
	return unsafe.Pointer(ret)
}

// sysMmap calls the mmap system call.  It is implemented in assembly.
func sysMmap(addr unsafe.Pointer, n uintptr, prot, flags, fd int32, off uint32) unsafe.Pointer

// cgoMmap calls the mmap function in the runtime/cgo package on the
// callCgoMmap calls the mmap function in the runtime/cgo package
// using the GCC calling convention.  It is implemented in assembly.
func callCgoMmap(addr unsafe.Pointer, n uintptr, prot, flags, fd int32, off uint32) uintptr

func callCgoMmapReserve(addr unsafe.Pointer, n uintptr) uintptr
