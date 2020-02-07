#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_var IN_PODMAN IN_PODMAN_NAME GOSRC

if [[ "$IN_PODMAN" == "true" ]]
then
    cd $GOSRC
    in_podman --rm $IN_PODMAN_NAME $0
else
    cd $GOSRC
    echo "Compiling buildah"
    showrun make $CROSS_TARGET ${CROSS_TARGET:+CGO_ENABLED=0}
    mkdir -p bin

    echo "Installing buildah"
    if [[ -z "$CROSS_TARGET" ]]
    then
        ln -v buildah bin/buildah
        showrun make install PREFIX=/usr
    else
        ln -v buildah.${CROSS_TARGET} bin/buildah
    fi
fi
