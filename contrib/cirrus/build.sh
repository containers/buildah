#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_vars IN_PODMAN IN_PODMAN_NAME GOSRC

remove_packaged_buildah_files

go version && go env

cd "$GOSRC"
if [[ "$IN_PODMAN" == "true" ]]
then
    in_podman --rm $IN_PODMAN_NAME $0
else
    echo "Compiling buildah (\$GOSRC=$GOSRC)"
    showrun make clean all

    echo "Installing buildah"
    mkdir -p bin
    showrun make install PREFIX=/usr
    showrun ./bin/buildah info
fi
