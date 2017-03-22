## buildah-from "March 2017"

## NAME
buildah from - Creates a new working container, either from scratch or based on the provided image name as a starting point.


## SYNOPSIS
**buildah** **from** --image *imagename* [*options* [...]] 

## DESCRIPTION
Creates a working container based upon the supplied image name.  If the supplied image name is "scratch" a new default container will be created.

## OPTIONS
**--name** *name*
Specifies a name to use for the newly created working container.

**--pull**
Pull the image if it is not present.  If this flag is not specified,the image will not be pulled regardless.

**--pull-always** 
Pull the image even if a version of the image is already present.

**--registry** *registry*
A prefix to prepend to the image name in order to pull the image.  Default value is "docker://"

**--signature-policy** *signaturepolicy*
The path of the for the signature policy file.  The default value is *PATH*.

**--mount**
The working container will be mounted when specified.

**--link** link
Path name of a symbolic link to create to the root directory of the container. 


## EXAMPLE
**buildah from --image imagename --pull --registry "myregistry://" --mount **

## SEE ALSO
buildah-list(1)
