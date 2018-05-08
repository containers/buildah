![buildah logo](https://cdn.rawgit.com/projectatomic/buildah/master/logos/buildah-logo_large.png)

# Useful Buildah links

**[Changelog](../CHANGELOG.md)**

**[Installation notes](../install.md)**

**[Troubleshooting Guide](../troubleshooting.md)**

**[Tutorials](../docs/tutorials/README.md)**

# Buildah Demos

The purpose of these demonstrations is twofold:

1. To help automate some of the tutorial material so that Buildah newcomers can walk through some of the concepts.
2. For Buildah enthusiasts and practitioners to use for demos at educational presentations - college classes, Meetups etc.

It is assumed that you have installed Buildah and Podman on your  machine.

    $ sudo yum -y install podman buildah

## Building from scratch demo 

filename: [`buildah-scratch-demo.sh`](https://github.com/projectatomic/buildah/demos/buildah-scratch-demo.sh)

This demo builds a container image from scratch. The container is going to inject a bash shell script and therefore requires the installation of coreutils and bash.

Please make sure you have installed Buildah and Podman. Also this demo uses Quay.io to push the image to that registry when it is completed. If you are not logged in then it will fail at that step and finish. If you wish to login to Quay.io before running the demo, then it will push to your repository successfully.

    $ sudo podman login quay.io

There are several variables you will want to set that are listed at the top of the script. The name for the container image, your quay.io username, your name, and the Fedora release number:

    demoimg=myshdemo
    quayuser=ipbabble
    myname=YourNameHere
    fedorarelease=28

## Buildah and Docker compatibility demo

Coming soon.


