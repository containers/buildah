[comment]: <> (***ATTENTION*** ***WARNING*** ***ALERT*** ***CAUTION*** ***DANGER***)
[comment]: <> ()
[comment]: <> (ANY changes made to this file, once committed/merged must)
[comment]: <> (be manually copy/pasted -in markdown- into the description)
[comment]: <> (field on Quay at the following locations:)
[comment]: <> ()
[comment]: <> (https://quay.io/repository/containers/buildah)
[comment]: <> (https://quay.io/repository/buildah/stable)
[comment]: <> (https://quay.io/repository/buildah/testing)
[comment]: <> (https://quay.io/repository/buildah/upstream)
[comment]: <> ()
[comment]: <> (***ATTENTION*** ***WARNING*** ***ALERT*** ***CAUTION*** ***DANGER***)

![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

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

  * `quay.io/containers/buildah:<version>` and `quay.io/buildah/stable:<version>` -
    These images are built daily. They are intended to contain an unchanging
    and stable version of buildah.  For the most recent `<version>` tags (`vX`,
    `vX.Y`, and `vX.Y.Z`) the image contents will be updated daily to incorporate
    (especially) security upgrades.  For build details, please [see the
    configuration file](stable/Dockerfile).
  * `quay.io/containers/buildah:latest` and `quay.io/buildah/stable:latest` -
    Built daily using the same Dockerfile as above.  The buildah version
    will remain the "latest" available in Fedora, however the other image
    contents may vary compared to the version-tagged images.
  * `quay.io/buildah/testing:latest` - This image is built daily, using the
    latest version of Buildah that was in the Fedora `updates-testing` repository.
    The image is Built with [the testing Dockerfile](testing/Dockerfile).
  * `quay.io/buildah/upstream:latest` - This image is built daily using the latest
    code found in this GitHub repository.  Due to the image changing frequently,
    it's not guaranteed to be stable or even executable.  The image is built with
    [the upstream Dockerfile](upstream/Dockerfile).  Note: The actual compilation
    of upstream buildah [occurs continuously in
    COPR](https://copr.fedorainfracloud.org/coprs/rhcontainerbot/podman-next/).


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
