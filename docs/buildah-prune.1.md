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

## EXAMPLE

buildah prune

buildah prune --force

## SEE ALSO

buildah(1), containers-registries.conf(5), containers-storage.conf(5)
