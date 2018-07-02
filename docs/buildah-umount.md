# buildah-umount "1" "March 2017" "buildah"

## NAME
buildah\-umount - Unmount the root file system on the specified working containers.

## SYNOPSIS
**buildah umount** [*options*]  [*container* ...]

## DESCRIPTION
Unmounts the root file system on the specified working containers.

## OPTIONS
**--all, -a**

All of the currently mounted containers will be unmounted.

## EXAMPLE

buildah umount containerID

buildah umount containerID1 containerID2 containerID3

buildah umount --all

## SEE ALSO
buildah(1)
