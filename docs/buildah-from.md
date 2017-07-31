## buildah-from "1" "March 2017" "buildah"

## NAME
buildah from - Creates a new working container, either from scratch or using a specified image as a starting point.

## SYNOPSIS
**buildah** **from** [*options* [...]] *imageName*

## DESCRIPTION
Creates a working container based upon the specified image name.  If the
supplied image name is "scratch" a new empty container is created.

## RETURN VALUE
The container ID of the container that was created.  On error, -1 is returned and errno is returned.

## OPTIONS

**--name** *name*

A *name* for the working container

**--pull**

Pull the image if it is not present.  If this flag is disabled (with
*--pull=false*) and the image is not present, the image will not be pulled.
Defaults to *true*.

**--pull-always**

Pull the image even if a version of the image is already present.

**--transport** *transport*

A prefix to prepend to the image name in order to pull the image.  The default
value is "docker://".  Note that no separator is implicitly added when the
values are combined.

**--signature-policy** *signaturepolicy*

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**--quiet**

If an image needs to be pulled from the registry, suppress progress output.

## EXAMPLE

buildah from imagename --pull --transport "docker://myregistry.example.com/"

buildah from docker://myregistry.example.com/imagename --pull

buildah from imagename --signature-policy /etc/containers/policy.json

buildah from imagename --pull-always --transport "docker://myregistry.example.com/" --name "mycontainer"

## SEE ALSO
buildah(1)
