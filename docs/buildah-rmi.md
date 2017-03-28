## buildah-rmi "March 2017"

## NAME
buildah rmi - Removes one or more images. 


## SYNOPSIS
**buildah** **rmi** **imageID(s)** 

## DESCRIPTION
Removes a locally stored image or images.  Multiple images are space separated.  If multiple images are passed to this command and the removal fails on one, the images following that image will not be removed.   

## EXAMPLE
**buildah rmi imageID **
**buildah rmi imageID1 imageID2 imageID3 **

## SEE ALSO
buildah(1)

