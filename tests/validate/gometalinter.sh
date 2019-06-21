#!/bin/bash
export PATH=tests/tools/build:${PATH}
if ! which gometalinter.v1 > /dev/null 2> /dev/null ; then
	echo gometalinter.v1 is not in PATH.
	echo Make sure to call this script from the project root.
	exit 1
fi
exec gometalinter.v1 \
	--enable-gc \
	--exclude='error return value not checked.*(Close|Log|Print).*\(errcheck\)$' \
	--exclude='.*_test\.go:.*error return value not checked.*\(errcheck\)$' \
	--exclude='declaration of.*err.*shadows declaration.*\(vetshadow\)$'\
	--exclude='duplicate of.*_test.go.*\(dupl\)$' \
	--exclude='vendor\/.*' \
	--exclude='test/tools\/.*' \
	--enable=unparam \
	--disable=gotype \
	--disable=gas \
	--disable=aligncheck \
	--cyclo-over=45 \
	--deadline=2000s \
	--tests "$@"
