#!/bin/bash

BUILDAH_BINARY=${BUILDAH_BINARY:-$(dirname ${BASH_SOURCE})/../buildah}
IMGTYPE_BINARY=${IMGTYPE_BINARY:-$(dirname ${BASH_SOURCE})/../imgtype}
TESTSDIR=${TESTSDIR:-$(dirname ${BASH_SOURCE})}
STORAGE_DRIVER=${STORAGE_DRIVER:-vfs}
PATH=$(dirname ${BASH_SOURCE})/..:${PATH}

function setup() {
	suffix=$(dd if=/dev/urandom bs=12 count=1 status=none | base64 | tr +/ABCDEFGHIJKLMNOPQRSTUVWXYZ _.abcdefghijklmnopqrstuvwxyz)
	TESTDIR=${BATS_TMPDIR}/tmp.${suffix}
	rm -fr ${TESTDIR}
	mkdir -p ${TESTDIR}/{root,runroot}
}

function starthttpd() {
	pushd ${2:-${TESTDIR}} > /dev/null
	cp ${TESTSDIR}/serve.go .
	go build serve.go
	HTTP_SERVER_PORT=$((RANDOM+32768))
	./serve ${HTTP_SERVER_PORT} ${1:-${BATS_TMPDIR}} &
	HTTP_SERVER_PID=$!
	popd > /dev/null
}

function stophttpd() {
	if test -n "$HTTP_SERVER_PID" ; then
		kill -HUP ${HTTP_SERVER_PID}
		unset HTTP_SERVER_PID
		unset HTTP_SERVER_PORT
	fi
	true
}

function teardown() {
	stophttpd
	rm -fr ${TESTDIR}
}

function createrandom() {
	dd if=/dev/urandom bs=1 count=${2:-256} of=${1:-${BATS_TMPDIR}/randomfile} status=none
}

function buildah() {
	${BUILDAH_BINARY} --debug --registries-conf ${TESTSDIR}/registries.conf --root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver ${STORAGE_DRIVER} "$@"
}

function imgtype() {
        ${IMGTYPE_BINARY} -root ${TESTDIR}/root -runroot ${TESTDIR}/runroot -storage-driver ${STORAGE_DRIVER} "$@"
}
