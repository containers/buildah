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
    # Nightly dependency-bump job: fetch latest versions of the
    # Big Three dependencies, and run full CI test suite. Notification
    # email will go out to monitor-list upon failure.
    if [[ "$CIRRUS_CRON" = "treadmill" ]]; then
        for pkg in common image/v5 storage; do
            echo "go mod edit --require containers/$pkg@main"
            go mod edit --require github.com/containers/$pkg@main
            make vendor
        done
        git add vendor
        # Show what changed.
        echo "git diff go.mod, then git diff --stat:"
        git diff go.mod
        git diff --stat
        env GIT_AUTHOR_NAME='No B. Dee'               \
            GIT_AUTHOR_EMAIL='nobody@example.com'     \
            GIT_COMMITTER_NAME='No B. Dee'            \
            GIT_COMMITTER_EMAIL='nobody@example.com'  \
            git commit -asm"Bump containers/common,image,storage"
    fi

    echo "Compiling buildah (\$GOSRC=$GOSRC)"
    showrun make clean all

    echo "Installing buildah"
    mkdir -p bin
    showrun make install PREFIX=/usr
    showrun ./bin/buildah info
fi
