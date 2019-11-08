#!/usr/bin/env bash

set -e

# N/B: Most development-related packages are pulled in automatically by
#      'build-essential' (on ubuntu), for Feodra the groups
#      '@C Development Tools and Libraries' and '@Development Tools'
#      are similar.

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
    libseccomp-dev
    libselinux-dev
    openssl
    podman
    netcat
    rsync
    scons
    vim
    wget
    yum-utils
    zlib1g-dev
    xz-utils
"

FEDORA_PACKAGES="
    bats
    btrfs-progs-devel
    bzip2
    containers-common
    device-mapper-devel
    findutils
    git
    glib2-devel
    glibc-static
    gnupg
    go-md2man
    golang
    gpgme-devel
    libassuan-devel
    libseccomp-devel
    make
    nmap-ncat
    ostree-devel
    podman
    rsync
    runc
    skopeo-containers
    wget
    xz
"

source $(dirname $0)/lib.sh

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
            for ppa in ppa:longsleep/golang-backports ppa:projectatomic/ppa; do
                timeout_attempt_delay_command 30 2 30 \
                    add-apt-repository --yes $ppa
            done
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
