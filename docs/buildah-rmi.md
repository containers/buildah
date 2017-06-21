## buildah-rmi "1" "March 2017" "buildah"

## NAME
buildah rmi - Removes one or more images.

## SYNOPSIS
**buildah** **rmi** **imageID [...]**

## DESCRIPTION
Removes one or more locally stored images.

## OPTIONS

**--force, -f**

Executing this command will stop all containers that are using the image and remove them from the system

## EXAMPLE

buildah rmi imageID

buildah rmi --force imageID

buildah rmi imageID1 imageID2 imageID3

## SEE ALSO
buildah(1)
