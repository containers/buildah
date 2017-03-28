## buildah-rm "March 2017"

## NAME
buildah rm - Removes one or more working containers. 


## SYNOPSIS
**buildah** **rm** **containerID(s)** 

## DESCRIPTION
Removes a working container or containers unmounting them if necessary.  Multiple containers are space separated.  If multiple containers are passed to this command and the removal fails on one, the containers following that container will not be removed. 

## EXAMPLE
**buildah delete containerID **
**buildah delete containerID1 containerID2 containerID3 **

## SEE ALSO
buildah(1)

