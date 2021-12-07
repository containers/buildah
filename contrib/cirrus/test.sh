#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_vars IN_PODMAN IN_PODMAN_NAME GOSRC 1

if [[ "$IN_PODMAN" == "true" ]]
then
    cd $GOSRC
    # Host build environment != container environment
    showrun make clean
    in_podman --rm $IN_PODMAN_NAME:latest $0 $1
else
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
            # Typically it's undesirable to install packages at runtime.
            # This test compares images built with the "latest" version
            # of docker, against images built with buildah. Runtime installs
            # are required to ensure the latest docker version is used.
            [[ "$OS_RELEASE_ID" == "ubuntu" ]] || \
                bad_os_id_ver
            warn "Installing previously downloaded/cached docker packages"
            ooe.sh dpkg -i \
                $PACKAGE_DOWNLOAD_DIR/containerd.io*.deb \
                $PACKAGE_DOWNLOAD_DIR/docker-ce*.deb

            systemctl enable --now docker
            showrun make test-conformance
            ;;
        integration)
            showrun make test-integration
            ;;
        *)
            die "First parameter to $(basename $0) not supported: '$1'"
            ;;
    esac
fi
