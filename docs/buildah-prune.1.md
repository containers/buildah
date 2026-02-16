# buildah-rmi "1" "Jan 2023" "buildah"

## NAME

buildah\-prune - Cleanup intermediate images as well as build and mount cache.

## SYNOPSIS

**buildah prune**

## DESCRIPTION

Cleanup intermediate images as well as build and mount cache.

## OPTIONS

**--all**, **-a**

All local images will be removed from the system that do not have containers using the image as a reference image.

**--force**, **-f**

This option will cause Buildah to remove all containers that are using the image before removing the image from the system.

**--tls-details** *path*

Path to a `containers-tls-details.yaml(5)` file, affecting TLS behavior throughout the program.

If not set, defaults to a reasonable default that may change over time (depending on system’s global policy,
version of the program, version of the Go language, and the like).

Users should generally not use this option unless they have a process to ensure that the configuration will be kept up to date.

## EXAMPLE

buildah prune

buildah prune --force

## SEE ALSO

buildah(1), containers-registries.conf(5), containers-storage.conf(5)
