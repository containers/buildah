buildah - a tool for building OCI images
========================================

[![Go Report Card](https://goreportcard.com/badge/github.com/nalind/buildah)](https://goreportcard.com/report/github.com/nalind/buildah)
[![Travis](https://travis-ci.org/nalind/buildah.svg?branch=master)](https://travis-ci.org/nalind/buildah)

Note: this package is pre-alpha, and will either move to an organization or be merged into a different project at some point.

The buildah package provides a command line tool which can be used to
* create a working container, either from scratch or using an image as a starting point
* mount the working container's root filesystem for manipulation
* umount the working container's root filesystem
* use the updated contents of the container's root filesystem as a filesystem layer to create a new image
* delete a working container

Future goals include:
* docs
* more CI tests
* additional CLI commands (build?)
