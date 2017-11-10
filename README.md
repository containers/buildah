![buildah logo](https://cdn.rawgit.com/projectatomic/buildah/master/logos/buildah.svg)

# [Buildah](https://www.youtube.com/embed/YVk5NgSiUw8) - a tool which facilitates building OCI container images
================================================================

[![Go Report Card](https://goreportcard.com/badge/github.com/projectatomic/buildah)](https://goreportcard.com/report/github.com/projectatomic/buildah)
[![Travis](https://travis-ci.org/projectatomic/buildah.svg?branch=master)](https://travis-ci.org/projectatomic/buildah)

Note: this package is in alpha, but is close to being feature-complete.

The Buildah package provides a command line tool which can be used to
* create a working container, either from scratch or using an image as a starting point
* create an image, either from a working container or via the instructions in a Dockerfile
* images can be built in either the OCI image format or the traditional upstream docker image format
* mount a working container's root filesystem for manipulation
* unmount a working container's root filesystem
* use the updated contents of a container's root filesystem as a filesystem layer to create a new image
* delete a working container or an image

**[Changelog](CHANGELOG.md)**

**Installation notes**

Prior to installing Buildah, install the following packages on your linux distro:
* make
* golang (Requires version 1.8.1 or higher.)
* bats
* btrfs-progs-devel
* bzip2
* device-mapper-devel
* git
* go-md2man
* gpgme-devel
* glib2-devel
* libassuan-devel
* ostree-devel
* runc (Requires version 1.0 RC4 or higher.)
* skopeo-containers

In Fedora, you can use this command:

```
 dnf -y install \
    make \
    golang \
    bats \
    btrfs-progs-devel \
    device-mapper-devel \
    glib2-devel \
    gpgme-devel \
    libassuan-devel \
    ostree-devel \
    git \
    bzip2 \
    go-md2man \
    runc \
    skopeo-containers
```

Then to install Buildah on Fedora follow the steps in this example:


```
  mkdir ~/buildah
  cd ~/buildah
  export GOPATH=`pwd`
  git clone https://github.com/projectatomic/buildah ./src/github.com/projectatomic/buildah
  cd ./src/github.com/projectatomic/buildah
  make
  make install
  buildah --help
```

In RHEL 7, ensure that you are subscribed to `rhel-7-server-rpms`,
`rhel-7-server-extras-rpms`, and `rhel-7-server-optional-rpms`, then
run this command:

```
 yum -y install \
    make \
    golang \
    bats \
    btrfs-progs-devel \
    device-mapper-devel \
    glib2-devel \
    gpgme-devel \
    libassuan-devel \
    ostree-devel \
    git \
    bzip2 \
    go-md2man \
    runc \
    skopeo-containers
```

The build steps for Buildah on RHEL are the same as Fedora, above.

In Ubuntu zesty and xenial, you can use this command:

```
  apt-get -y install software-properties-common
  add-apt-repository -y ppa:alexlarsson/flatpak
  add-apt-repository -y ppa:gophers/archive
  apt-add-repository -y ppa:projectatomic/ppa
  apt-get -y -qq update
  apt-get -y install bats btrfs-tools git libapparmor-dev libdevmapper-dev libglib2.0-dev libgpgme11-dev libostree-dev libseccomp-dev libselinux1-dev skopeo-containers go-md2man
  apt-get -y install golang-1.8
```
Then to install Buildah on Ubuntu follow the steps in this example:

```
  mkdir ~/buildah
  cd ~/buildah
  export GOPATH=`pwd`
  git clone https://github.com/projectatomic/buildah ./src/github.com/projectatomic/buildah
  cd ./src/github.com/projectatomic/buildah
  PATH=/usr/lib/go-1.8/bin:$PATH make runc all TAGS="apparmor seccomp"
  make install
  buildah --help
```
Buildah uses `runc` to run commands when `buildah run` is used, or when `buildah build-using-dockerfile`
encounters a `RUN` instruction, so you'll also need to build and install a compatible version of
[runc](https://github.com/opencontainers/runc) for Buildah to call for those cases.

## Commands
| Command                                              | Description                                                                                          |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| [buildah-add(1)](/docs/buildah-add.md)               | Add the contents of a file, URL, or a directory to the container.                                    |
| [buildah-bud(1)](/docs/buildah-bud.md)               | Build an image using instructions from Dockerfiles.                                                  |
| [buildah-commit(1)](/docs/buildah-commit.md)         | Create an image from a working container.                                                            |
| [buildah-config(1)](/docs/buildah-config.md)         | Update image configuration settings.                                                                 |
| [buildah-containers(1)](/docs/buildah-containers.md) | List the working containers and their base images.                                                   |
| [buildah-copy(1)](/docs/buildah-copy.md)             | Copies the contents of a file, URL, or directory into a container's working directory.               |
| [buildah-from(1)](/docs/buildah-from.md)             | Creates a new working container, either from scratch or using a specified image as a starting point. |
| [buildah-images(1)](/docs/buildah-images.md)         | List images in local storage.                                                                        |
| [buildah-inspect(1)](/docs/buildah-inspect.md)       | Inspects the configuration of a container or image.                                                  |
| [buildah-mount(1)](/docs/buildah-mount.md)           | Mount the working container's root filesystem.                                                       |
| [buildah-push(1)](/docs/buildah-push.md)             | Copies an image from local storage.                                                                  |
| [buildah-rm(1)](/docs/buildah-rm.md)                 | Removes one or more working containers.                                                              |
| [buildah-rmi(1)](/docs/buildah-rmi.md)               | Removes one or more images.                                                                          |
| [buildah-run(1)](/docs/buildah-run.md)               | Run a command inside of the container.                                                               |
| [buildah-tag(1)](/docs/buildah-tag.md)               | Add an additional name to a local image.                                                             |
| [buildah-umount(1)](/docs/buildah-umount.md)         | Unmount a working container's root file system.                                                      |
| [buildah-version(1)](/docs/buildah-version.md)       | Display the Buildah Version Information             |

**Future goals include:**
* more CI tests
* additional CLI commands (?)
