# buildah-rm "1" "March 2017" "buildah"

## NAME
buildah\-rm - Removes one or more working containers.

## SYNOPSIS
**buildah rm** *container* ...

## DESCRIPTION
Removes one or more working containers, unmounting them if necessary.

## OPTIONS

**--all, -a**

All Buildah containers will be removed.  Buildah containers are denoted with an '*' in the 'BUILDER' column listed by the command 'buildah containers'.

## EXAMPLE

buildah rm containerID

buildah rm containerID1 containerID2 containerID3

buildah rm --all

## SEE ALSO
buildah(1)
