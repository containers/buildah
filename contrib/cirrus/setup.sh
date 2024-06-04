#!/usr/bin/env bash

set -e

# N/B: In most (but not all) cases, these packages will already be installed
# in the VM image at build-time (from libpod repo.).  Running package install
# again here, ensures that all cases are covered, and there is never any
# expectation mismatch.
source $(dirname $0)/lib.sh

req_env_vars OS_RELEASE_ID OS_RELEASE_VER GOSRC IN_PODMAN_IMAGE CIRRUS_CHANGE_TITLE

msg "Running df."
df -hT

msg "Disabling git repository owner-check system-wide."
# Newer versions of git bark if repo. files are unexpectedly owned.
# This mainly affects rootless and containerized testing.  But
# the testing environment is disposable, so we don't care.=
git config --system --add safe.directory $GOSRC

# Support optional/draft testing using latest/greatest
# podman-next COPR packages.  This requires a draft PR
# to ensure changes also pass CI w/o package updates.
if [[ "$OS_RELEASE_ID" =~ "fedora" ]] && \
   [[ "$CIRRUS_CHANGE_TITLE" =~ CI:NEXT ]]
then
    # shellcheck disable=SC2154
    if [[ "$CIRRUS_PR_DRAFT" != "true" ]]; then
        die "Magic 'CI:NEXT' string can only be used on DRAFT PRs"
    fi

    showrun dnf copr enable rhcontainerbot/podman-next -y
    showrun dnf upgrade -y
fi

msg "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
cd $GOSRC
case "$OS_RELEASE_ID" in
    fedora)
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
    debian)
        if [[ "$1" == "conformance" ]]; then
            msg "Installing previously downloaded/cached Docker packages"
            dpkg -i \
                $PACKAGE_DOWNLOAD_DIR/containerd.io*.deb \
                $PACKAGE_DOWNLOAD_DIR/docker-ce*.deb
        fi
        ;;
    *)
        bad_os_id_ver
        ;;
esac

# Required to be defined by caller: Are we testing as root or a regular user
case "$PRIV_NAME" in
    root)
        if [[ "$TEST_FLAVOR" = "sys" ]]; then
            # Used in local image-scp testing
            setup_rootless
        fi
        ;;
    rootless)
        # load kernel modules since the rootless user has no permission to do so
        modprobe ip6_tables || :
        modprobe ip6table_nat || :
        setup_rootless
        ;;
    *) die_unknown PRIV_NAME
esac

# Previously, golang was not installed
source $(dirname $0)/lib.sh

echo "Configuring /etc/containers/registries.conf"
mkdir -p /etc/containers
echo -e "[registries.search]\nregistries = ['docker.io', 'registry.fedoraproject.org', 'quay.io']" | tee /etc/containers/registries.conf

show_env_vars

if [[ -z "$CONTAINER" ]]; then
    # Discovered reemergence of BFQ scheduler bug in kernel 5.8.12-200
    # which causes a kernel panic when system is under heavy I/O load.
    # Previously discovered in F32beta and confirmed fixed. It's been
    # observed in F31 kernels as well.  Deploy workaround for all VMs
    # to ensure a more stable I/O scheduler (elevator).
    echo "mq-deadline" > /sys/block/sda/queue/scheduler
    warn "I/O scheduler: $(cat /sys/block/sda/queue/scheduler)"
fi

execute_local_registry  # checks for existing port 5000 listener

if [[ "$IN_PODMAN" == "true" ]]
then
    req_env_vars IN_PODMAN_IMAGE IN_PODMAN_NAME
    echo "Setting up image to use for \$IN_PODMAN=true testing"
    cd $GOSRC
    in_podman $IN_PODMAN_IMAGE $0
    showrun podman commit $IN_PODMAN_NAME $IN_PODMAN_NAME
    showrun podman rm -f $IN_PODMAN_NAME
fi
