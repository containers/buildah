#!/bin/bash

set -e

if [[ "$SPC" != "true" ]]
then
    echo "This script is intended to be executed in an SPC,"
    echo "by run_ci_tests.sh. Using it otherwise may result"
    echo "in unpleasant side-effects."
    exit 1
fi

echo
echo "Build Environment:"
env

set +x

echo "Updating image and deps..."
dnf -y update --best --allowerasing
dnf -y install dnf-plugins-core
dnf -y copr enable lsm5/container-diff
dnf -y install autoconf automake btrfs-progs-devel \
   buildah \
   container-diff device-mapper-devel golang go-md2man \
   git glibc-static gpgme-devel hostname iproute \
   iputils libassuan-devel libseccomp-static make \
   moby-engine openssl

# build buildah binary in fedora and run tests
echo "Cleanup buildah repo and build again in fedora..."
make clean
mv vendor src
mkdir -p $(pwd)/_build/src/github.com/projectatomic
ln -s $(pwd) $(pwd)/_build/src/github.com/projectatomic/buildah
make GOPATH=$(pwd)/_build:$(pwd) all TAGS="seccomp containers_image_ostree_stub"
GOPATH=$(pwd)/_build:$(pwd) go test -c -tags "seccomp `./btrfs_tag.sh` `./libdm_tag.sh` `./ostree_tag.sh` `./selinux_tag.sh`" ./cmd/buildah
tmp=$(mktemp -d); mkdir $tmp/root $tmp/runroot; PATH="$PATH" ./buildah.test -test.v -root $tmp/root -runroot $tmp/runroot -storage-driver vfs -signature-policy $(pwd)/tests/policy.json -registries-conf $(pwd)/tests/registries.conf

echo "docker login to local registry..."
echo testpassword | docker login localhost:5000 --username testuser --password-stdin

echo "docker build dockerfile..."
docker build -f hack/Dockerfile -t docker-test-image .

echo "buildah bud dockerfile..."
./buildah --registries-conf tests/registries.conf bud --file hack/Dockerfile -t buildah-test-image .

echo "buildah tag buildah-test-image..."
./buildah tag buildah-test-image localhost:5000/buildah-test-image

echo "buildah push buildah-test-image..."
./buildah push --cert-dir /home/travis/auth --tls-verify=false --authfile /root/.docker/config.json buildah-test-image localhost:5000/buildah-test-image

echo "docker pull buildah-test-image..."
docker pull localhost:5000/buildah-test-image

echo "Running container-diff..."
container-diff diff --type=rpm daemon://localhost:5000/buildah-test-image daemon://docker-test-image

