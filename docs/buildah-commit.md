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

**--cert-dir** *path*

Use certificates at *path* (*.crt, *.cert, *.key) to connect to the registry

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--disable-compression, -D**

Don't compress filesystem layers when building the image.


**--format**

Control the format for the image manifest and configuration data.  Recognized
formats include *oci* (OCI image-spec v1.0, the default) and *docker* (version
2, using schema format 2 for the manifest).

**--quiet**

When writing the output image, suppress progress output.

**--rm**
Remove the container and its content after committing it to an image.
Default leaves the container and its content in place.

**--signature-policy**

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)

## EXAMPLE

This example saves an image based on the container.
 `buildah commit containerID`

This example saves an image named newImageName based on the container.
 `buildah commit --rm containerID newImageName`


This example saves an image based on the container disabling compression.
 `buildah commit --disable-compression containerID`

This example saves an image named newImageName based on the container disabling compression.
 `buildah commit --disable-compression containerID newImageName`

This example commits the container to the image on the local registry while turning off tls verification.
 `buildah commit --tls-verify=false containerID docker://localhost:5000/imageId`

This example commits the container to the image on the local registry using credentials and certificates for authentication.
 `buildah commit --cert-dir ~/auth  --tls-verify=true --creds=username:password containerID docker://localhost:5000/imageId`

## SEE ALSO
buildah(1)
