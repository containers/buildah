## buildah-commit "1" "March 2017" "buildah"

## NAME
buildah commit - Create an image from a working container.

## SYNOPSIS
**buildah** **commit** [*options* [...]] **containerID** [**imageName**]

## DESCRIPTION
Writes a new image using the specified container's read-write layer and if it
is based on an image, the layers of that image.  If an image name is not
specified, an ID is assigned, but no name is assigned to the image.

## OPTIONS

**-c**, **--change**=[]
   Apply specified Dockerfile instructions while committing the image
   Supported Dockerfile instructions: `CMD`|`ENTRYPOINT`|`ENV`|`EXPOSE`|`ONBUILD`|`USER`|`VOLUME`|`WORKDIR`

**--disable-compression, -D**

Don't compress filesystem layers when building the image.

**-m**, **--message**=""
   Set commit message for container image

**--signature-policy**

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**--quiet**

When writing the output image, suppress progress output.

**--format**

Control the format for the image manifest and configuration data.  Recognized
formats include *oci* (OCI image-spec v1.0, the default) and *docker* (version
2, using schema format 2 for the manifest).

**--rm**
Remove the container and its content after committing it to an image.
Default leaves the container and its content in place.

## EXAMPLE

buildah commit containerID

buildah commit --rm containerID newImageName

buildah commit --disable-compression --signature-policy '/etc/containers/policy.json' containerID

buildah commit --disable-compression --signature-policy '/etc/containers/policy.json' containerID newImageName

## SEE ALSO
buildah(1)
