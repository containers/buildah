#!/usr/bin/env bash

set -e

# N/B: In most (but not all) cases, these packages will already be installed
# in the VM image at build-time (from libpod repo.).  Running package install
# again here, ensures that all cases are covered, and there is never any
# expectation mismatch.
source $(dirname $0)/lib.sh

req_env_var OS_RELEASE_ID OS_RELEASE_VER GOSRC IN_PODMAN_IMAGE

echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
cd $GOSRC
case "$OS_RELEASE_ID" in
    fedora)
        # This can be removed when the kernel bug fix is included in Fedora
        if [[ $OS_RELEASE_VER -le 32 ]] && [[ -z "$CONTAINER" ]]; then
            warn "Switching io scheduler to 'deadline' to avoid RHBZ 1767539"
            warn "aka https://bugzilla.kernel.org/show_bug.cgi?id=205447"
            echo "mq-deadline" | sudo tee /sys/block/sda/queue/scheduler > /dev/null
            echo -n "IO Scheduler set to: "
            $SUDO cat /sys/block/sda/queue/scheduler
        fi

        # Not executing IN_PODMAN container
        if [[ -z "$CONTAINER" ]]; then
            warn "Adding secondary testing partition & growing root filesystem"
            bash $SCRIPT_BASE/add_second_partition.sh

            warn "TODO: Add (for htpasswd) to VM images (in libpod repo)"
            dnf install -y httpd-tools
        fi

        warn "Hard-coding podman to use crun"
	cat > /etc/containers/containers.conf <<EOF
[engine]
runtime="crun"
EOF

        # Executing tests in a container requires SELinux boolean set on the host
        if [[ "$IN_PODMAN" == "true" ]]
        then
            showrun setsebool -P container_manage_cgroup true
        fi
        ;;
    ubuntu)
        warn "TODO: Add to VM images (in libpod repo)"
        $SHORT_APTGET update
        $SHORT_APTGET install apache2-utils
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
echo -e "[registries.search]\nregistries = ['docker.io', 'registry.fedoraproject.org', 'quay.io']" | tee /etc/containers/registries.conf

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
