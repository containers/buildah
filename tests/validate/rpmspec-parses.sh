#!/bin/bash
if ! rpmspec --version > /dev/null ; then
	echo Warning: rpmspec not found in \$PATH, can not verify syntax of contrib/rpm/buildah.spec
	exit 0
fi
set -e
specfile=$(dirname ${BASH_SOURCE})/../../contrib/rpm/buildah.spec
rpmspec -q ${specfile}
if rpmspec -q ${specfile} 2>&1 | grep -qi error ; then
	exit 1
fi
