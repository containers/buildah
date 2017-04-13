## buildah-mount "1" "March 2017" "buildah"

## NAME
buildah mount - Mount a working container's root filesystem.

## SYNOPSIS
**buildah** **mount** **containerID**

## DESCRIPTION
Mounts the specified container's root file system in a location which can be
accessed from the host, and returns its location.

## RETURN VALUE
The location of the mounted file system.  On error an empty string and errno is
returned.

## EXAMPLE

buildah mount containerID

## SEE ALSO
buildah(1)
