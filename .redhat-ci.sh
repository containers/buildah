#!/bin/bash
set -xeuo pipefail

export GOPATH=$HOME/gopath
export PATH=$HOME/gopath/bin:$PATH
export GOSRC=$HOME/gopath/src/github.com/projectatomic/buildah

(mkdir -p $GOSRC && cd /code && cp -r . $GOSRC)

dnf install -y \
  make \
  golang \
  bats \
  btrfs-progs-devel \
  device-mapper-devel \
  gpgme-devel \
  libassuan-devel \
  bzip2

make -C $GOSRC
$GOSRC/tests/test_runner.sh
