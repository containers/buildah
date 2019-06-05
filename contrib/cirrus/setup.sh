#!/bin/bash

set -e

# N/B: Most development-related packages are pulled in automaticly by
#      'build-essential' (on ubuntu), for Feodra the groups
#      '@C Development Tools and Libraries' and '@Development Tools'
#      are similar.

# TODO: FIXME: Arbitrary list of packages, adjustment likely required.
UBUNTU_PACKAGES="
    aufs-tools
    bats
    bzip2
    coreutils
    curl
    git
    go-md2man
    golang
    libdevmapper-dev
    libglib2.0-dev
    libgpgme11-dev
    libostree-dev
    libseccomp-dev
    libselinux-dev
    openssl
    rsync
    scons
    vim
    wget
    yum-utils
    zlib1g-dev
"

# TODO: FIXME: Slightly arbitrary list of packages, adjustment possibly required.
FEDORA_PACKAGES="
    bats
    btrfs-progs-devel
    containers-common
    device-mapper-devel
    git
    glib2-devel
    go-md2man
    golang
    gpgme-devel
    libassuan-devel
    libseccomp-devel
    ostree-devel
    podman
    runc
    skopeo-containers
    wget
"

source $(dirname $0)/lib.sh

show_env_vars

install_ooe

echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"  # STUB: Add VM setup instructions here
cd $GOSRC
case "$OS_REL_VER" in
    fedora-*)
        # When the fedora repos go down, it tends to last quite a while :(
        timeout_attempt_delay_command 120s 3 120s dnf install -y \
             '@C Development Tools and Libraries' '@Development Tools' \
            $FEDORA_PACKAGES
        ;;
    ubuntu-*)
        $SHORT_APTGET update
        $LONG_APTGET upgrade
        if [[ "$OS_RELEASE_VER" == "18" ]]
        then
            echo "(Enabling newer golang on Ubuntu LTS version)"
            $SHORT_APTGET install software-properties-common
            $SHORT_APTGET update
            timeout_attempt_delay_command 30 2 30 \
                add-apt-repository --yes ppa:longsleep/golang-backports
        fi
        $LONG_APTGET install \
            build-essential \
            $UBUNTU_PACKAGES
        ;;
    *)
        bad_os_id_ver
        ;;
esac

# Previously, golang was not installed
source $(dirname $0)/lib.sh

echo "Installing buildah tooling"
showrun make install.tools
