buildah - a tool for building OCI images
========================================

[![Go Report Card](https://goreportcard.com/badge/github.com/projectatomic/buildah)](https://goreportcard.com/report/github.com/projectatomic/buildah)
[![Travis](https://travis-ci.org/projectatomic/buildah.svg?branch=master)](https://travis-ci.org/projectatomic/buildah)

Note: this package is in alpha.

The buildah package provides a command line tool which can be used to
* create a working container, either from scratch or using an image as a starting point
* mount the working container's root filesystem for manipulation
* unmount the working container's root filesystem
* use the updated contents of the container's root filesystem as a filesystem layer to create a new image
* delete a working container

Future goals include:
* docs
* more CI tests
* additional CLI commands (build?)
