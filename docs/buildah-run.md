
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

**--tty**

By default a pseudo-TTY is allocated only when buildah's standard input is
attached to a pseudo-TTY. Setting the `--tty` option to `true` will cause a
pseudo-TTY to be allocated inside the container. Setting the `--tty` option to
`false` will prevent the pseudo-TTY from being allocated.

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime.

**--runtime-flag** *flag*

Adds global flags for the container runtime. To list the supported flags, please
consult manpages of your selected container runtime (`runc` is the default
runtime, the manpage to consult is `runc(8)`)

**--volume, -v** *source*:*destination*:*flags*

Bind mount a location from the host into the container for its lifetime.

NOTE: End parsing of options with the `--` option, so that you can pass other 
options to the command inside of the container

## EXAMPLE

buildah run containerID -- ps -auxw

buildah run containerID --runtime-flag --no-new-keyring -- ps -auxw

## SEE ALSO
buildah(1)
