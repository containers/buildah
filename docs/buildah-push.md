## buildah-push"1" "June 2017" "buildah"

## NAME
buildah push - Push an image from local storage to elsewhere.

## SYNOPSIS
**buildah** **push** [*options* [...]] **imageID** [**destination**]

## DESCRIPTION
Pushes an image from local storage to a specified destination, decompressing
and recompessing layers as needed.

## OPTIONS

**--disable-compression, -D**

Don't compress copies of filesystem layers which will be pushed.

**--signature-policy**

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**--quiet**

When writing the output image, suppress progress output.

## EXAMPLE

buildah push imageID dir:/path/to/image

buildah push imageID oci-layout:/path/to/layout

buildah push imageID docker://registry/repository:tag

buildah push imageID docker-daemon:repository:tag

## SEE ALSO
buildah(1)
