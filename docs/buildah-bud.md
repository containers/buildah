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

**--authfile** *path*

Path of the authentication file. Default is ${XDG_RUNTIME\_DIR}/containers/auth.json, which is set using `podman login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

**--build-arg** *arg=value*

Specifies a build argument and its value, which will be interpolated in
instructions read from the Dockerfiles in the same way that environment
variables are, but which will not be added to environment variable list in the
resulting image's configuration.

**--cert-dir** *path*

Use certificates at *path* (*.crt, *.cert, *.key) to connect to the registry.
Default certificates directory is _/etc/containers/certs.d_.

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**-f, --file** *Dockerfile*

Specifies a Dockerfile which contains instructions for building the image,
either a local file or an **http** or **https** URL.  If more than one
Dockerfile is specified, *FROM* instructions will only be accepted from the
first specified file.

If a build context is not specified, and at least one Dockerfile is a
local file, the directory in which it resides will be used as the build
context.

**--format**

Control the format for the built image's manifest and configuration data.
Recognized formats include *oci* (OCI image-spec v1.0, the default) and
*docker* (version 2, using schema format 2 for the manifest).

**--pull**

Pull the image if it is not present.  If this flag is disabled (with
*--pull=false*) and the image is not present, the image will not be pulled.
Defaults to *true*.

**--pull-always**

Pull the image even if a version of the image is already present.

**-q, --quiet**

Suppress output messages which indicate which instruction is being processed,
and of progress when pulling images from a registry, and when writing the
output image.

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime, which will be used to run
commands specified by the **RUN** instruction.

**--runtime-flag** *flag*

Adds global flags for the container rutime. To list the supported flags, please
consult manpages of your selected container runtime (`runc` is the default
runtime, the manpage to consult is `runc(8)`).
Note: Do not pass the leading `--` to the flag. To pass the runc flag `--log-format json`
to buildah bud, the option given would be `--runtime-flag log-format=json`.

**--signature-policy** *signaturepolicy*

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**-t, --tag** *imageName*

Specifies the name which will be assigned to the resulting image if the build
process completes successfully.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)

## EXAMPLE

buildah bud .

buildah bud -f Dockerfile.simple .

buildah bud -f Dockerfile.simple -f Dockerfile.notsosimple

buildah bud -t imageName .

buildah bud --tls-verify=true -t imageName -f Dockerfile.simple

buildah bud --tls-verify=false -t imageName .

buildah bud --runtime-flag log-format=json .

buildah bud --runtime-flag debug .

buildah bud --authfile /tmp/auths/myauths.json --cert-dir ~/auth --tls-verify=true --creds=username:password -t imageName -f Dockerfile.simple

## SEE ALSO
buildah(1), podman-login(1), docker-login(1)
