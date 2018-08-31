#!/bin/bash
export PATH=${GOPATH%%:*}/bin:${PATH}
if ! which gometalinter.v1 > /dev/null 2> /dev/null ; then
	echo gometalinter.v1 is not installed.
	echo Try installing it with \"make install.tools\" or with
	echo \"go get -u gopkg.in/alecthomas/gometalinter.v1\"
	echo \"gometalinter.v1 --install --vendored-linters\"
	exit 1
fi
exec gometalinter.v1 \
	--enable-gc \
	--exclude='error return value not checked.*(Close|Log|Print).*\(errcheck\)$' \
	--exclude='.*_test\.go:.*error return value not checked.*\(errcheck\)$' \
	--exclude='declaration of.*err.*shadows declaration.*\(vetshadow\)$'\
	--exclude='duplicate of.*_test.go.*\(dupl\)$' \
	--exclude='vendor\/.*' \
	--enable=unparam \
	--disable=gotype \
	--disable=gas \
	--disable=aligncheck \
	--cyclo-over=45 \
	--deadline=2000s \
	--tests "$@"
