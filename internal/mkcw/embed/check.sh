#!/usr/bin/env bash
expected="This image is designed to be run as a confidential workload using libkrun."
cd $(dirname ${BASH_SOURCE[0]})
for GOARCH in amd64 arm64 ppc64le s390x ; do
	make -C ../../.. internal/mkcw/embed/entrypoint_$GOARCH
	case $GOARCH in
		amd64) QEMUARCH=x86_64;;
		arm64) QEMUARCH=aarch64;;
		ppc64le|s390x) QEMUARCH=$GOARCH;;
	esac
	actual="$(qemu-$QEMUARCH ./entrypoint_$GOARCH 2>&1)"
	if test "$actual" != "$expected" ; then
		echo unexpected error from entrypoint_$GOARCH: "$actual"
		exit 1
	fi
done
