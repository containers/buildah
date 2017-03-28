## buildah-containers "March 2017"

## NAME
buildah containers - List the working containers and their base images. 


## SYNOPSIS
**buildah** **containers** [*options* [...]] 

## DESCRIPTION
Lists containers which appear to be buildah working containers, their names and IDs, and the names and IDs of the images from which they were initialized.

## OPTIONS

**--noheading, -n **
Omit the table headings from the listing of containers.

**--notruncate**
Do not truncate output.

**--quiet, -q **
Displays only the container image id's.

## EXAMPLE
**buildah containers **
**buildah containers --quiet **
**buildah containers -q --noheading --notruncate **

## SEE ALSO
buildah(1)

