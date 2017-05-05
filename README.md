buildah - a tool for building OCI images
========================================

[![Go Report Card](https://goreportcard.com/badge/github.com/projectatomic/buildah)](https://goreportcard.com/report/github.com/projectatomic/buildah)
[![Travis](https://travis-ci.org/projectatomic/buildah.svg?branch=master)](https://travis-ci.org/projectatomic/buildah)

Note: this package is in alpha.

The buildah package provides a command line tool which can be used to
* create a working container, either from scratch or using an image as a starting point
* mount a working container's root filesystem for manipulation
* unmount a working container's root filesystem
* use the updated contents of a container's root filesystem as a filesystem layer to create a new image
* delete a working container or an image

**Installation notes**

Prior to installing buildah, install the following packages on your linux distro:
* make
* golang
* bats
* btrfs-progs-devel 
* device-mapper-devel 
* gpgme-devel 
* libassuan-devel 
* git 
* bzip2
* go-md2man 

In Fedora, you can use this command:

```
 dnf -y install \ 
    make \ 
    golang \ 
    bats \ 
    btrfs-progs-devel \ 
    device-mapper-devel \ 
    gpgme-devel \ 
    libassuan-devel \ 
    git \ 
    bzip2 \
    go-md2man
```

Then to install buildah follow the steps in this example: 

```
  mkdir ~/buildah
  cd ~/buildah
  export GOPATH=`pwd` 
  git clone https://github.com/projectatomic/buildah ./src/github.com/projectatomic/buildah 
  cd ./src/github.com/projectatomic/buildah 
  make 
  make install
  ./buildah --help
```

## Commands
| Command               | Description |
| --------------------- | --------------------------------------------------- |
| buildah-add(1)        | Add the contents of a file, URL, or a directory to the container. |
| buildah-bud(1)        | Build an image using instructions from Dockerfiles. |
| buildah-commit(1)     | Create an image from a working container. |
| buildah-config(1)     | Update image configuration settings. |
| buildah-containers(1) | List the working containers and their base images. |
| buildah-copy(1)       | Copies the contents of a file, URL, or directory into a container's working directory. |
| buildah-from(1)       | Creates a new working container, either from scratch or using a specified image as a starting point. |
| buildah-images(1)     | List images in local storage. |
| buildah-mount(1)      | Mount the working container's root filesystem. |
| buildah-rm(1)         | Removes one or more working containers. |
| buildah-rmi(1)        | Removes one or more images. |
| buildah-run(1)        | Run a command inside of the container. |
| buildah-umount(1)     | Unmount a working container's root file system. |

**Future goals include:**
* more CI tests
* additional CLI commands (?)
