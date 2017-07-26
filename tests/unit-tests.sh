#!/bin/bash
TOP=$(dirname ${BASH_SOURCE})/..
cd "$TOP"
packages=$(find -name "*_test.go" | xargs -r -n 1 dirname | sort -u)
exec go test $LDFLAGS $BUILDFLAGS "$@" $packages
