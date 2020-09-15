#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_var IN_PODMAN IN_PODMAN_NAME GOSRC 1

if [[ "$IN_PODMAN" == "true" ]]
then
    cd $GOSRC
    # Host build environment != container environment
    showrun make clean
    in_podman --rm $IN_PODMAN_NAME:latest $0 $1
elif [[ -z "$CROSS_TARGET" ]]
then
    cd $GOSRC

    showrun make
    showrun make install.tools

    case $1 in
        validate)
            showrun ooe.sh git remote add upstream "$CIRRUS_REPO_CLONE_URL"
            showrun ooe.sh git remote update
            if [[ -n "$CIRRUS_PR" ]]; then
                echo "Validating a PR"
                export GITVALIDATE_EPOCH="$CIRRUS_BASE_SHA"
            elif [[ -n "$CIRRUS_TAG" ]]; then
                echo "Refusing to validating a Tag"
                return 0
            else
                echo "Validating a Branch"
                export GITVALIDATE_EPOCH="$CIRRUS_LAST_GREEN_CHANGE"
            fi
            echo "Linting & Validating from ${GITVALIDATE_EPOCH:-default EPOCH}"
            showrun make lint LINTFLAGS="--deadline=20m --color=always -j1"
            showrun make validate
            ;;
        unit)
            showrun make test-unit
            ;;
        conformance)
            case "$OS_RELEASE_ID" in
            fedora)
                warn "Installing moby-engine"
                dnf install -y moby-engine
                systemctl enable --now docker
                ;;
            ubuntu)
                warn "Installing docker.io"
                $LONG_APTGET install docker.io
                systemctl enable --now docker
                ;;
            *)
                bad_os_id_ver
                ;;
            esac
            showrun make test-conformance
            ;;
        integration)
            showrun make test-integration
            ;;
        *)
            die 1 "First parameter to $(basename $0) not supported: '$1'"
            ;;
    esac
else
    echo "Testing a cross-compiled $CROSS_TARGET target not possible on this platform"
fi
