# buildah-rmi "1" "March 2017" "buildah"

## NAME
buildah\-rmi - Removes one or more images.

## SYNOPSIS
**buildah rmi** *image* ...

## DESCRIPTION
Removes one or more locally stored images.

## LIMITATIONS
If the image was pushed to a directory path using the 'dir:' transport
the rmi command can not remove the image.  Instead standard file system
commands should be used.
If _imageID_ is a name, but does not include a registry name, buildah will attempt to find and remove an image named using the registry name *localhost*, if no such image is found, it will search for the intended image by attempting to expand the given name using the names of registries provided in the system's registries configuration file, registries.conf.

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

## Files

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

## SEE ALSO
buildah(1), registries.conf(5)
