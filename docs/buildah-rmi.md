# buildah-rmi "1" "March 2017" "buildah"

## NAME
buildah rmi - Removes one or more images.

## SYNOPSIS
**buildah** **rmi** **imageID [...]**

## DESCRIPTION
Removes one or more locally stored images.

## LIMITATIONS
If the image was pushed to a directory path using the 'dir:' transport
the rmi command can not remove the image.  Instead standard file system
commands should be used.

## OPTIONS

**--all, -a**

All local images will be removed from the system that do not have containers using the image as a reference image.

**--prune, -p**

All local images will be removed from the system that do not have a tag and do not have a child image pointing to them.

**--force, -f**

This option will cause Buildah to remove all containers that are using the image before removing the image from the system.

## EXAMPLE

buildah rmi imageID

buildah rmi --all

buildah rmi --all --force

buildah rmi --prune

buildah rmi --force imageID

buildah rmi imageID1 imageID2 imageID3

## SEE ALSO
buildah(1)
