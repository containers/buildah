#!/bin/bash

set -e

# This script can be run manually - assuming password-less sudo access
# and a docker-daemon running.  To play in the SPC, just
# `export SPCCMD=bash` beforehand.

SPCCMD="${SPCCMD:-./hack/spc_build.sh}"
DISTRO="${DISTRO:-fedora}"
FQIN="${DISTRO}:latest"
sudo docker pull $FQIN &

REPO_DIR=$(realpath "$(dirname $0)/../")  # assume parent directory of 'hack'
REPO_NAME=$(basename $(git rev-parse --show-toplevel))
# In Travis $PWD == $TRAVIS_BUILD_DIR == $HOME/$REPO_OWNER/$REPO_NAME
TRAVIS_BUILD_DIR="/root/$REPO_NAME"
WORKDIR="$TRAVIS_BUILD_DIR"

# Volume-mounting the repo into the SPC makes a giant mess of permissions
# on the host.  This really sucks for developers, so make a copy for use
# in the SPC separate from the host, throw it away when this script exits.
echo
echo "Making temporary copy of $REPO_DIR that"
echo "will appear in SPC as $WORKDIR"
TMP_SPC_REPO_COPY=$(mktemp -p '' -d ${REPO_NAME}_XXXXXX)
trap "sudo rm -rf $TMP_SPC_REPO_COPY" EXIT
/usr/bin/rsync -avz --recursive --links --delete-after --quiet \
               --delay-updates --whole-file --safe-links \
               --perms --times "${REPO_DIR}/" "${TMP_SPC_REPO_COPY}/"\
               --include=.git &

SPC_ARGS="--interactive --rm --privileged --ipc=host --pid=host --net=host"

VOL_ARGS="-v $TMP_SPC_REPO_COPY:$WORKDIR
          -v /run:/run -v /etc/localtime:/etc/localtime
          -v /var/log:/var/log -v /sys/fs/cgroup:/sys/fs/cgroup
          -v /var/lib/containers:/var/lib/containers
          -v /var/run/docker.sock:/var/run/docker.sock
          --tmpfs /tmp:rw,nosuid,nodev,exec,relatime,mode=1777,size=50%
          --workdir $WORKDIR"

ENV_ARGS="-e HOME=/root -e SHELL=/bin/bash -e SPC=true -e DISTRO=$DISTRO -e WORKDIR=$WORKDIR"

echo
echo "Preparing to run \$SPCMD=$SPCCMD in a $DISTRO SPC."
echo "Override either for a different experience."
wait  # for backgrounded processes to finish
echo
set -x
sudo docker run -t $SPC_ARGS $VOL_ARGS $ENV_ARGS $TRAVIS_ENV $FQIN $SPCCMD
