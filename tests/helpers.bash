#!/bin/bash

BUILDAH_BINARY=${BUILDAH_BINARY:-$(dirname ${BASH_SOURCE})/../buildah}
TESTSDIR=${TESTSDIR:-$(dirname ${BASH_SOURCE})}

function setup() {
	suffix=$(dd if=/dev/urandom bs=12 count=1 status=none | base64)
	TOPDIR=${BATS_TMPDIR}/${suffix}
	rm -fr ${TOPDIR}
	mkdir -p ${TOPDIR}/{root,runroot}
	REPO=${TOPDIR}/root
}

function teardown() {
	rm -fr ${TOPDIR}
}

function createrandom() {
	dd if=/dev/urandom bs=1 count=${2:-256} of=${1:-${BATS_TMPDIR}/randomfile} status=none
}

function buildah() {
	${BUILDAH_BINARY} --debug --root ${TOPDIR}/root --runroot ${TOPDIR}/runroot --storage-driver vfs "$@"
}
