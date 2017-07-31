## buildah-bud "1" "April 2017" "buildah"

## NAME
buildah bud - Build an image using instructions from Dockerfiles.

## SYNOPSIS
**buildah** **bud | build-using-dockerfile** [*options* [...]] [**context**]

## DESCRIPTION
Builds an image using instructions from one or more Dockerfiles and a specified
build context directory.  The build context directory can be specified as the
**http** or **https** URL of an archive which will be retrieved and extracted
to a temporary location.

## OPTIONS

**-f, --file** *Dockerfile*

Specifies a Dockerfile which contains instructions for building the image,
either a local file or an **http** or **https** URL.  If more than one
Dockerfile is specified, *FROM* instructions will only be accepted from the
first specified file.

If a build context is not specified, and at least one Dockerfile is a
local file, the directory in which it resides will be used as the build
context.

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

**--build-arg** *arg=value*

Specifies a build argument and its value, which will be interpolated in
instructions read from the Dockerfiles in the same way that environment
variables are, but which will not be added to environment variable list in the
resulting image's configuration.

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime, which will be used to run
commands specified by the **RUN** instruction.

**--runtime-flag** *flag*

Adds global flags for the container rutime.

**-t, --tag** *imageName*

Specifies the name which will be assigned to the resulting image if the build
process completes successfully.

**--format**

Control the format for the built image's manifest and configuration data.
Recognized formats include *oci* (OCI image-spec v1.0, the default) and
*docker* (version 2, using schema format 2 for the manifest).

**--quiet**

Suppress output messages which indicate which instruction is being processed,
and of progress when pulling images from a registry, and when writing the
output image.

## EXAMPLE

buildah bud .

buildah bud -f Dockerfile.simple .

buildah bud -f Dockerfile.simple -f Dockerfile.notsosimple

buildah bud -t imageName .

buildah bud -t imageName -f Dockerfile.simple

## SEE ALSO
buildah(1)
