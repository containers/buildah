# Container Tools Guide

## Introduction

The purpose of this guide is to list a number of related Open-source projects that are available
on [GitHub.com](https://github.com) that operate on
[Open Container Initiative](https://www.opencontainers.org/) (OCI) images and containers.  This
guide will give a high level explanation of the related container tools and will explain a bit
on how they interact amongst each other.

The tools are:

* [Buildah](https://github.com/containers/buildah)
* [CRI-O](https://github.com/kubernetes-sigs/cri-o)
* [Podman](https://github.com/containers/podman)
* [Skopeo](https://github.com/containers/skopeo)

## Buildah

The Buildah project provides a command line tool that be used to create an OCI or traditional Docker
image format image and to then build a working container from the image.  The container can be mounted
and modified and then an image can be saved based on the updated container.

## CRI-O

CRI-O [Website](http://cri-o.io/)

The CRI-O project provides an integration path between OCI conformant runtimes and kubelet.
Specifically, it implements the Kubelet
[Container Runtime Interface (CRI)](https://github.com/kubernetes/community/blob/master/contributors/devel/container-runtime-interface.md)
using OCI conformant runtimes.   The scope of CRI-O is tied to the scope of the CRI.

At a high level CRI-O supports multiple image formats including the existing Docker image format,
multiple means to download images including trust & image verification, container image and lifecycle
management, monitoring, logging and resource isolation as required by CRI.

## Podman

[Podman](https://github.com/containers/podman) allows for full management of a container's lifecycle from creation
through removal.  It supports multiple image formats including both the Docker and OCI image formats.  Support for
pods is provided allowing pods to manage groups of containers together.  Podman also supports trust and image
verification when pulling images along with resource isolation of containers and pods.

## Skopeo

Skopeo is a command line tool that performs a variety of operations on container images and image repositories.
Skopeo can work on either OCI or Docker images.  Skopeo can be used to copy images from and to various
container storage mechanisms including container registries.  Skopeo also allows you to inspect an image
showing its layers without requiring that the image be pulled.  Skopeo also allows you to delete an image
from a repository.  When required by the repository, Skopeo can pass appropriate certificates and credentials
for authentication.


## Buildah and Podman relationship

Buildah and Podman are two complementary Open-source projects that are available on
most Linux platforms and both projects reside at [GitHub.com](https://github.com)
with Buildah [here](https://github.com/containers/buildah) and
Podman [here](https://github.com/containers/podman).  Both Buildah and Podman are
command line tools that work on OCI images and containers.  The two projects
differentiate in their specialization.

Buildah specializes in building OCI images.  Buildah's commands replicate all
of the commands that are found in a Dockerfile. Buildah’s goal is also to
provide a lower level coreutils interface to build images, allowing people to build
containers without requiring a Dockerfile.  The intent with Buildah is to allow other
scripting languages to build container images, without requiring a daemon.

Podman specializes in all of the commands and functions that help you to maintain and modify
OCI images, such as pulling and tagging.  It also allows you to create, run, and maintain containers
created from those images.

A major difference between Podman and Buildah is their concept of a container.  Podman
allows users to create "traditional containers" where the intent of these containers is
to be long lived.  While Buildah containers are really just created to allow content
to be added back to the container image.   An easy way to think of it is the
`buildah run` command emulates the RUN command in a Dockerfile while the `podman run`
command emulates the `docker run` command in functionality.  Because of this you
cannot see Podman containers from within Buildah or vice versa.

In short Buildah is an efficient way to create OCI images  while Podman allows
you to manage and maintain those images and containers in a production environment using
familiar container cli commands.

Some of the commands between the projects overlap:

* build
The `podman build` and `buildah build` commands have significant overlap as Podman borrows large pieces of the `podman build` implementation from Buildah.

* run
The `buildah run` and `podman run` commands are similar but different.  As explained above Podman and Buildah have a different concept of a container.  An easy way to think of it is the `buildah run` command emulates the RUN command in a Dockerfile while the `podman run` command emulates the `docker run` command in functionality.  As Buildah and Podman have somewhat different concepts of containers, you can not see Podman containers from within Buildah or vice versa.

* pull, push
These commands are basically the same between the two and either could be used.

* commit
Commit works differently because of the differences in `containers`.  You cannot commit a Podman container from Buildah nor a Buildah container from Podman.

* tag, rmi, images
These commands are basically the same between the two and either could be used.

* rm
This command appears to be equivalent on the surface, but they differ due to the underlying storage differences of containers
between the two projects.  Given that, Buildah containers can not be removed with a Podman command and Podman containers
can not be removed with a Buildah command.

* mount
Mount command is similar for both in that you can mount the container image and modify content in it, which will be saved to an image when you commit.

In short Buildah is an efficient way to create OCI images  while Podman allows
you to manage and maintain those images and containers in a production environment using
familiar container cli commands.

