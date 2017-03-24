## buildah-from "March 2017"

## NAME
buildah from - Creates a container image based on the supplied image name. If the supplied image name is "scratch" a new default empty container image will be created. 


## SYNOPSIS
**buildah** **from** --image *imagename* [*options* [...]] 

## DESCRIPTION
Creates a working container based upon the supplied image name.  If the supplied image name is "scratch" a new default container will be created.

## OPTIONS

**--pull**
Pull the image if it is not present.  If this flag is not specified,the image will not be pulled regardless.  Defaults to TRUE.

**--pull-always** 
Pull the image even if a version of the image is already present.

**--registry** *registry*
A prefix to prepend to the image name in order to pull the image.  Default value is "docker://"

**--signature-policy** *signaturepolicy*
The path of the for the signature policy file.  The default location is */etc/containers/policy.json* and changing to another file is not recommended.

**--mount**
The working container will be mounted when specified.

**--link** link
Path name of a symbolic link to create to the root directory of the container. 


## EXAMPLE
**buildah from --image imagename --pull --registry "myregistry://" --mount **
**buildah from --image imagename --mount --link ~/mycontainerroot --signature-policy /etc/containers/policy.json **
**buildah from --image imagename --pull-always --registry "myregistry://" **

## SEE ALSO
buildah-list(1)
