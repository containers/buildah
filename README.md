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
    bzip2 
```

Then to install buildah follow the steps in this example: 

```
  mkdir ~/buildah
  cd ~/buildah
  export GOPATH=`pwd` 
  git clone https://github.com/projectatomic/buildah ./src/github.com/projectatomic/buildah 
  cd ./src/github.com/projectatomic/buildah 
  make 
  ./buildah --help
```

**Future goals include:**
* docs
* more CI tests
* additional CLI commands (?)
