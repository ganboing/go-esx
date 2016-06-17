#!/bin/bash

set -e

if [ ! -f bootstrap.bash ]; then
	echo 'bootstrap-esx.bash must be run from $GOROOT/src' 1>&2
	exit 1
fi

ESX_TOOLS=$PWD/../esx_tools

export GOROOT_BOOTSTRAP="${GOROOT_BOOTSTRAP:-/usr/lib/go}"
export GOOS=esx
export GOARCH=amd64
export CGO_ENABLED=1
export CC_FOR_TARGET=x86_64-vmk-linux-gnu-gcc
export CXX_FOR_TARGET=x86_64-vmk-linux-gnu-g++
export LD_FOR_TARGET=x86_64-vmk-linux-gnu-ld
export ESX_SYSROOT=$ESX_TOOLS/esx_sysroot
export PATH=$PATH:$ESX_TOOLS:$ESX_TOOLS/esx_toolchain/usr/bin

$PWD/bootstrap.bash
