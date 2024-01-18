#!/usr/bin/env bash

set -e

# N/B: In most (but not all) cases, these packages will already be installed
# in the VM image at build-time (from libpod repo.).  Running package install
# again here, ensures that all cases are covered, and there is never any
# expectation mismatch.
source $(dirname $0)/lib.sh

req_env_vars OS_RELEASE_ID OS_RELEASE_VER GOSRC IN_PODMAN_IMAGE

echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
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

        # At the end of 2023/beginning of 2024, golang needs to be 1.17
        # in order to address GHSA-45x7-px36-x8w8 CVE-2023-48795
        # Ref: https://github.com/containers/buildah/pull/5245
        # This being on a release-branch, replacing the CI VMs is
        # ill-advised, and risks giving false-positive testing results.
        showrun dnf erase -y golang
        cd /usr/local
        showrun curl -Ssl -O https://dl.google.com/go/go1.17.13.linux-amd64.tar.gz
        showrun tar -xf go1.17.13.linux-amd64.tar.gz
        cd -
        # This needs to be set in every test-execution context. Hack it in here
        # since lib.sh always loads this before running any tests.
        echo 'export PATH=$PATH:/usr/local/go/bin' | tee -a /etc/automation_environment
        export PATH=$PATH:/usr/local/go/bin
        showrun go version
        showrun go env
        ;;
    ubuntu)
        if [[ "$1" == "conformance" ]]; then
            msg "Installing previously downloaded/cached packages"
            ooe.sh dpkg -i \
                $PACKAGE_DOWNLOAD_DIR/containerd.io*.deb \
                $PACKAGE_DOWNLOAD_DIR/docker-ce*.deb

            # At the time of this comment, Ubuntu is using systemd-resolved
            # which interfears badly with conformance testing.  Some tests
            # need to run dnsmasq on port 53.
            if [[ -r "/run/systemd/resolve/resolv.conf" ]]; then
                msg "Disabling systemd-resolved service"
                systemctl stop systemd-resolved.service
                cp /run/systemd/resolve/resolv.conf /etc/
            fi
        fi
        ;;
    *)
        bad_os_id_ver
        ;;
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
