#!/usr/bin/env bash

set -exo pipefail

uname -r

rpm -q \
    aardvark-dns \
    buildah \
    buildah-tests \
    conmon \
    container-selinux \
    containers-common \
    crun \
    netavark \
    systemd

bats /usr/share/buildah/test/system
