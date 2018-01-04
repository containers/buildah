## buildah-rm "1" "March 2017" "buildah"

## NAME
buildah rm - Removes one or more working containers.

## SYNOPSIS
**buildah** **rm** **containerID [...]**

## DESCRIPTION
Removes one or more working containers, unmounting them if necessary.

## OPTIONS

**--all, -a**

All containers will be removed.

## EXAMPLE

buildah rm containerID

buildah rm containerID1 containerID2 containerID3

buildah rm --all

## SEE ALSO
buildah(1)
