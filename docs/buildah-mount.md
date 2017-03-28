## buildah-mount "March 2017"

## NAME
buildah mount - Mount the working container's root filesystem.


## SYNOPSIS
**buildah** **mount** **containerID** 

## DESCRIPTION
Mount the container's root file system in a location which can be accessed from the host, and returns the location.

## RETURN VALUE
The location of the mounted file system.  On error an empty string and errno is returned. 

## EXAMPLE
**buildah mount containerID **
**mountFS=buildah mount containerID **

## SEE ALSO
buildah(1)

