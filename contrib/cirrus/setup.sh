#!/usr/bin/env bash

set -e

# N/B: In most (but not all) cases, these packages will already be installed
# in the VM image at build-time (from libpod repo.).  Running package install
# again here, ensures that all cases are covered, and there is never any
# expectation mismatch.
source $(dirname $0)/lib.sh

req_env_var OS_RELEASE_ID OS_RELEASE_VER GOSRC FEDORA_BUILDAH_PACKAGES UBUNTU_BUILDAH_PACKAGES IN_PODMAN_IMAGE

echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
cd $GOSRC
case "$OS_RELEASE_ID" in
    fedora)
        if [[ "$OS_RELEASE_VER" == "31" ]]; then
            warn "Switching io schedular to deadline to avoid RHBZ 1767539"
            warn "aka https://bugzilla.kernel.org/show_bug.cgi?id=205447"
            echo "mq-deadline" > /sys/block/sda/queue/scheduler
            cat /sys/block/sda/queue/scheduler
        fi

        warn "Adding secondary testing partition & growing root filesystem"
        bash $SCRIPT_BASE/add_second_partition.sh

        # Executing tests in a container requires SELinux boolean set on the host
        if [[ "$IN_PODMAN" == "true" ]]
        then
            setsebool -P container_manage_cgroup true
        fi
        ;;
    ubuntu)
        $SHORT_APTGET update
        $LONG_APTGET install ${UBUNTU_BUILDAH_PACKAGES[@]}
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
