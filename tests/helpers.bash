#!/bin/bash

BUILDAH_BINARY=${BUILDAH_BINARY:-$(dirname ${BASH_SOURCE})/../buildah}
TESTSDIR=${TESTSDIR:-$(dirname ${BASH_SOURCE})}

function setup() {
	suffix=$(dd if=/dev/urandom bs=12 count=1 status=none | base64 | tr +/ _.)
	TESTDIR=${BATS_TMPDIR}/tmp.${suffix}
	rm -fr ${TESTDIR}
	mkdir -p ${TESTDIR}/{root,runroot}
	REPO=${TESTDIR}/root
}

function teardown() {
	rm -fr ${TESTDIR}
}

function createrandom() {
	dd if=/dev/urandom bs=1 count=${2:-256} of=${1:-${BATS_TMPDIR}/randomfile} status=none
}

function buildah() {
	${BUILDAH_BINARY} --debug --root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver vfs "$@"
}
