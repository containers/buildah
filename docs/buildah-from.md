## buildah-from "March 2017"

## NAME
buildah from - Creates a new working container, either from scratch or using a specified image as a starting point. 


## SYNOPSIS
**buildah** **from** *imagename* [*options* [...]] 

## DESCRIPTION
Creates a working container based upon the supplied image name.  If the supplied image name is "scratch" a new default container will be created.

## RETURN VALUE
The container id of the container that was created.  On error, -1 is returned and errno is returned. 

## OPTIONS

**--mount** 
Mount the working container printing the mount point upon successful completion.

**--name** *name*
A *name* for the working container

**--pull**
Pull the image if it is not present.  If this flag is not specified,the image will not be pulled regardless.  Defaults to TRUE.

**--pull-always** 
Pull the image even if a version of the image is already present.

**--registry** *registry*
A prefix to prepend to the image name in order to pull the image.  Default value is "docker://"

**--signature-policy** *signaturepolicy*
The path of the for the signature policy file.  The default location is */etc/containers/policy.json* and changing to another file is not recommended.


## EXAMPLE
**buildah from imagename --pull --registry "myregistry://" --mount **
**buildah from imagename --signature-policy /etc/containers/policy.json **
**buildah from imagename --pull-always --registry "myregistry://" --name "mycontainer" **

## SEE ALSO
buildah(1)

