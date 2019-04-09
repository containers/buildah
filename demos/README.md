![buildah logo](https://cdn.rawgit.com/containers/buildah/master/logos/buildah-logo_large.png)

# Buildah Demos

The purpose of these demonstrations is twofold:

1. To help automate some of the tutorial material so that Buildah newcomers can walk through some of the concepts.
2. For Buildah enthusiasts and practitioners to use for demos at educational presentations - college classes, Meetups etc.

It is assumed that you have installed Buildah and Podman on your machine.

    $ sudo yum -y install podman buildah

For the Docker compatibility demo you will also need to install Docker.

    $ sudo yum -y install docker

Replace `yum` with `dnf` if required.

## Building from scratch demo

filename: [`buildah-scratch-demo.sh`](buildah-scratch-demo.sh)

This demo builds a container image from scratch. The container is going to inject a bash shell script and therefore requires the installation of coreutils and bash.

Please make sure you have installed Buildah and Podman. Also this demo uses Quay.io to push the image to that registry when it is completed. If you are not logged in then it will fail at that step and finish. If you wish to login to Quay.io before running the demo, then it will push to your repository successfully.

    $ sudo buildah login quay.io

There are several variables you will want to set that are listed at the top of the script. The name for the container image, your quay.io username, your name, and the Fedora release number:

    demoimg=myshdemo
    quayuser=UserNameHere
    myname=YourNameHere
    distrorelease=28
    pkgmgr=dnf   # switch to yum if using yum

## Buildah and Docker compatibility demo

filename: [`docker-compatibility-demo.sh`](docker-compatibility-demo.sh)

This demo builds an nginx container image using Buildah. It modifies the homepage and commits the image. The container is tested using `podman run` and then stopped. The Docker daemon is then started and the image is pushed to the Docker repository. The container is started using `docker run` and tested.

There are several variables you will want to set that are listed at the top of the script. The name for the container image, your quay.io username, your name, and the Fedora release number:

    demoimg=dockercompatibilitydemo
    quayuser=UsernameHere
    myname=YourNameHear
    distro=fedora
    distrorelease=28
    pkgmgr=dnf   # switch to yum if using yum

## Buildah build using Docker demo

filename: [`docker-bud-demo.sh`](buildah-bud-demo.sh)

This demo builds an nginx container image using Buildah with. Buildah's `buildah-using-docker`, or `bud` option, provides a mechanism for using existing Dockerfiles to build the container image. This image is the same as the image in the Docker compatibility demo (at time of creating this README). The container is tested using `podman run` and then stopped. The Docker daemon is then started and the image is pushed to the Docker repository. The container is started using `docker run` and tested.

There are several variables you will want to set that are listed at the top of the script. The name for the container image, your quay.io username, your name, and the Fedora release number:

    demoimg=buildahbuddemo
    quayuser=UsernameHere
    myname=YourNameHear
    distro=fedora
    distrorelease=28
    pkgmgr=dnf   # switch to yum if using yum
