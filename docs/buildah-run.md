## buildah-run "1" "March 2017" "buildah"

## NAME
buildah run - Run a command inside of the container.

## SYNOPSIS
**buildah** **run** **containerID** [*options* [...] --] **command**

## DESCRIPTION
Launches a container and runs the specified command in that container using the
container's root filesystem as a root filesystem, using configuration settings
inherited from the container's image or as specified using previous calls to
the *buildah config* command.

## OPTIONS

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime.

**--runtime-flag** *flag*

Adds global flags for the container rutime.

**--volume, -v** *source*:*destination*:*flags*

Bind mount a location from the host into the container for its lifetime.

## EXAMPLE

buildah run containerID 'ps -auxw'

buildah run containerID --runtime-flag --no-new-keyring 'ps -auxw'

## SEE ALSO
buildah(1)
