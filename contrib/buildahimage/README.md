![buildah logo](https://cdn.rawgit.com/containers/buildah/master/logos/buildah-logo_large.png)

# buildahimage

## Overview

This directory contains the Dockerfiles necessary to create the buildahimage container
images that are housed on quay.io under the buildah account.  All repositories where
the images live are public and can be pulled without credentials.  These container images are secured and the
resulting containers can run safely with privileges within the container.

The container images are built using the latest Fedora and then Buildah is installed into them.
The PATH in the container images is set to the default PATH provided by Fedora.  Also, the
ENTRYPOINT and the WORKDIR variables are not set within these container images, as such they
default to `/`.

The container images are:

  * quay.io/containers/buildah - This image is built using the latest stable version of Buildah in a Fedora based container.  Built with buildahimage/stable/Dockerfile.
  * quay.io/buildah/stable - This image is built using the latest stable version of Buildah in a Fedora based container.  Built with buildahimage/stable/Dockerfile.
  * quay.io/buildah/upstream - This image is built using the latest code found in this GitHub repository.  When someone creates a commit and pushes it, the image is created.  Due to that the image changes frequently and is not guaranteed to be stable.  Built with buildahimage/upstream/Dockerfile.
  * quay.io/buildah/testing - This image is built using the latest version of Buildah that is or was in updates testing for Fedora.  At times this may be the same as the stable image.  This container image will primarily be used by the development teams for verification testing when a new package is created.  Built with buildahimage/testing/Dockerfile.
  * quay.io/buildah/stable:version - This image is built 'by hand' using a Fedora based container.  An RPM is first pulled from the [Fedora Updates System](https://bodhi.fedoraproject.org/) and the image is built from there.  For more details, see the Containerfile used to build it, buildahimage/stablebyhand/Containerfile.buildahstable

## Sample Usage

Although not required, it is suggested that [Podman](https://github.com/containers/podman) be used with these container images.

```
podman pull docker://quay.io/buildah/stable:latest

podman run stable buildah version

# Create a directory on the host to mount the container's
# /var/lib/container directory to so containers can be
# run within the container.
mkdir /var/lib/mycontainer

# Run the image detached using the host's network in a container name
# buildahctr, turn off label and seccomp confinement in the container
# and then do a little shell hackery to keep the container up and running.
podman run --detach --name=buildahctr --net=host --security-opt label=disable --security-opt seccomp=unconfined --device /dev/fuse:rw -v /var/lib/mycontainer:/var/lib/containers:Z  stable sh -c 'while true ;do sleep 100000 ; done'

podman exec -it  buildahctr /bin/sh

# Now inside of the container

buildah from alpine

buildah images

exit
```

**Note:** If you encounter a `fuse: device not found` error when running the container image, it is likely that
the fuse kernel module has not been loaded on your host system.  Use the command `modprobe fuse` to load the
module and then run the container image.  To enable this automatically at boot time, you can add a configuration
file to `/etc/modules.load.d`.  See `man modules-load.d` for more details.

## Compatible with old versions
For compatibility reasons, some Linux distributions with a kernel version of 3.10.0 and less will not work when using the stable image that is based on fedora:32. This centos7 image can be used to work on those distributions.

Changes between it and stable:
- change base image from `fedora:latest` to `centos:7`
- remove `--exclude container-selinux` when installing `buildah` and `fuse-overlayfs`, for details pls check [here](https://bugzilla.redhat.com/show_bug.cgi?id=1806044)
- update the sed logic of storage.conf (the final result is the same, this change just changes the methodology)

> for more details of the compatible discussion pls check the [issue: 2393](https://github.com/containers/buildah/issues/2393)
