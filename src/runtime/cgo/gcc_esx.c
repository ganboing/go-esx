// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build cgo

// +build esx

#include <stdint.h>

extern void* mmapfix_reserve(void* addr, uintptr_t length);

void* x_cgo_mmap_reserve(void* addr, uintptr_t length) {

	return mmapfix_reserve(addr, length);
}
