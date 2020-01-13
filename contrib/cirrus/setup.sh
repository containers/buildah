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
    runc
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
        # Filling up cache is very slow and failures can last quite a while :(
        $LONG_DNFY install \
             '@C Development Tools and Libraries' '@Development Tools' \
            $FEDORA_PACKAGES
        # Executing tests in a container requires SELinux boolean set on the host
        if [[ "$IN_PODMAN" == "true" ]]
        then
            setsebool -P container_manage_cgroup true
        fi
        ;;
    ubuntu-*)
        $SHORT_APTGET update
        $LONG_APTGET upgrade
        $SHORT_APTGET install software-properties-common
        ppas=(ppa:projectatomic/ppa)
        if [[ "$OS_RELEASE_VER" == "18" ]]
        then
            ppas+=(ppa:longsleep/golang-backports)  # newer golang
        fi
        for ppa in ${ppas[@]}; do
            timeout_attempt_delay_command 30 2 30 \
                add-apt-repository --yes $ppa
        done
        echo "Configuring/Instaling deps from Open build server"
        . /etc/os-release
        echo "deb http://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/x${NAME}_${VERSION_ID}/ /" > /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list
        showrun curl -L -o /tmp/Release.key "https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable/x${NAME}_${VERSION_ID}/Release.key"
        apt-key add - < /tmp/Release.key
        $SHORT_APTGET update
        $LONG_APTGET install \
            build-essential \
            $UBUNTU_PACKAGES
        if [[ "$OS_RELEASE_VER" -le "19" ]]; then
            echo "Replacing old/buggy version of bats with newer package"
            # An IFS related bug in the stock bats version will causes failures in 'run'
            # Use newer/static package from https://launchpad.net/bats/trunk
            apt-get -qq remove --yes bats
            cd /tmp
            BATS_URL='http://launchpadlibrarian.net/438140887/bats_1.1.0+git104-g1c83a1b-1_all.deb'
            curl -L -O "$BATS_URL"
            apt-get -qq install --yes /tmp/$(basename $BATS_URL)
            cd -
        fi
        ;;
    *)
        bad_os_id_ver
        ;;
esac

# Previously, golang was not installed
source $(dirname $0)/lib.sh

X="export GPG_TTY=/dev/null"
echo "Setting $X in /etc/environment for proper GPG functioning under automation"
echo "$X" >> /etc/environment

echo "Configuring /etc/containers/registries.conf"
mkdir -p /etc/containers
echo -e "[registries.search]\nregistries = ['docker.io', 'quay.io']" | tee /etc/containers/registries.conf

show_env_vars

if [[ -z "$CROSS_TARGET" ]]
then
    remove_storage_mountopt  # workaround issue 1945 (remove when resolved)

    execute_local_registry  # checks for existing port 5000 listener

    if [[ "$IN_PODMAN" == "true" ]]
    then
        req_env_var IN_PODMAN_IMAGE IN_PODMAN_NAME
        echo "Setting up image to use for \$IN_PODMAN=true testing"
        cd $GOSRC
        in_podman $IN_PODMAN_IMAGE $0
        showrun podman commit $IN_PODMAN_NAME $IN_PODMAN_NAME
        showrun podman rm -f $IN_PODMAN_NAME
    fi
fi
