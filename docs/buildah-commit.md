## buildah-commit "March 2017"

## NAME
buildah commit - Create an image from a working container. 


## SYNOPSIS
**buildah** **commit** **containerID** [*command options* [...]] 

## DESCRIPTION
Writes a new image using the container's read-write layer and if it is based on an image, the layers of that image are written.

## OPTIONS

**--disable-compression**
Don't compress the layers.

**--signature-policy
Pathname of the signature policy file to use.  This option is generally not recommended for use. 

## EXAMPLE
**buildah commit containerID **
**buildah commit containerID --disable-compression --signature-policy '/etc/containers/policy.json' **

## SEE ALSO
buildah(1)

