#!/bin/bash
set -xeuo pipefail

export GOPATH=$HOME/gopath
export PATH=$HOME/gopath/bin:$PATH
export GOSRC=$HOME/gopath/src/github.com/projectatomic/buildah

(mkdir -p $GOSRC && cd /code && cp -r . $GOSRC)

dnf install -y \
  bats \
  btrfs-progs-devel \
  bzip2 \
  device-mapper-devel \
  findutils \
  git \
  golang \
  gpgme-devel \
  libassuan-devel \
  make \
  which

# Red Hat CI adds a merge commit, for testing, which fails the
# short-commit-subject validation test, so tell git-validate.sh to only check
# up to, but not including, the merge commit.
export GITVALIDATE_TIP=$(cd $GOSRC; git log -2 --pretty='%H' | tail -n 1)
make -C $GOSRC install.tools all validate
$GOSRC/tests/test_runner.sh
