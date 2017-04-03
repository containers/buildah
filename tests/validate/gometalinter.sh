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
	--exclude='error return value not checked.*(Close|Log|Print).*\(errcheck\)$' \
	--exclude='.*_test\.go:.*error return value not checked.*\(errcheck\)$' \
	--exclude='duplicate of.*_test.go.*\(dupl\)$' \
	--exclude='vendor\/.*' \
	--disable=gotype \
	--disable=gas \
	--disable=aligncheck \
	--cyclo-over=40 \
	--deadline=120s \
	--tests "$@"
