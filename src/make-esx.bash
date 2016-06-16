#!/bin/bash

set -e

if [ ! -f make.bash ]; then
	echo 'make-esx.bash must be run from $GOROOT/src' 1>&2
	exit 1
fi

export GOROOT_BOOTSTRAP="${GOROOT_BOOTSTRAP:-/usr/lib/go}"
export GOOS=esx
export GOARCH=amd64
export CGO_ENABLED=1
export CC_FOR_TARGET=$PWD/../gcc.esx
export CXX_FOR_TARGET=$PWD/../g++.esx
export LD_FOR_TARGET=$PWD/../ld.esx
export ESX_SYSROOT=$PWD/../esx_sysroot
export PATH=$PATH:$PWD/../esx_toolchain/usr/bin

patchlist=$'\
libc libc-2.12.2.so mmap 3
libc libc-2.12.2.so munmap 0
libc libc-2.12.2.so mremap 3
ld ld-2.12.2.so mmap 3
ld ld-2.12.2.so munmap 0'

#OFFSET_H=$PWD/runtime/cgo/mmap_wrapper_esx_amd64.h
#rm ${OFFSET_H}

#while read line; do
#	$PWD/esx-patch-gen $line >> ${OFFSET_H}
#done <<< "$patchlist"

$PWD/make.bash
