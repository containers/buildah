#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_var IN_PODMAN IN_PODMAN_NAME GOSRC

remove_packaged_buildah_files

go version && go env

cd "$GOSRC"
if [[ "$IN_PODMAN" == "true" ]]
then
    in_podman --rm $IN_PODMAN_NAME $0
else
    echo "Compiling buildah (\$GOSRC=$GOSRC)"
    showrun make clean ${CROSS_TARGET:-all} ${CROSS_TARGET:+CGO_ENABLED=0}

    echo "Installing buildah"
    mkdir -p bin
    if [[ -z "$CROSS_TARGET" ]]
    then
        showrun make install PREFIX=/usr
        showrun ./bin/buildah info
    fi
fi
