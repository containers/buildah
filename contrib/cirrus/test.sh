#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_vars IN_PODMAN IN_PODMAN_NAME GOSRC 1

# shellcheck disable=SC2154
if [[ "$PRIV_NAME" == "rootless" ]] && [[ "$UID" -eq 0 ]]; then
    # Remove /var/lib/cni, it is not required for rootless cni.
    # We have to test that it works without this directory.
    # https://github.com/containers/podman/issues/10857
    rm -rf /var/lib/cni

    # change permission of go src and cache directory
    # so rootless user can access it
    chown -R $ROOTLESS_USER:root /var/tmp/go
    chmod -R g+rwx /var/tmp/go

    req_env_vars ROOTLESS_USER
    msg "Re-executing test through ssh as user '$ROOTLESS_USER'"
    msg "************************************************************"
    set -x
    exec ssh $ROOTLESS_USER@localhost \
            -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
            -o CheckHostIP=no $GOSRC/$SCRIPT_BASE/test.sh $1
    # Does not return!
elif [[ "$UID" -ne 0 ]]; then
    # Load important env. vars written during setup.sh (run as root)
    # call to setup_rootless()
    source /home/$ROOTLESS_USER/ci_environment
fi
# else: not running rootless, do nothing special

msg "Test-time env. var. definitions (filtered):"
show_env_vars

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
            [[ "$OS_RELEASE_ID" == "debian" ]] || \
                bad_os_id_ver

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
