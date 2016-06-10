// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

func VMKForkExec(filepath *byte, argv, envp **byte, wdfd int32, initfds *int32, initfdslength uint32,
	uid, gid int32, detached bool, pid *uint32) Errno
