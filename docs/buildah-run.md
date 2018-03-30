# buildah-run "1" "March 2017" "buildah"

## NAME
buildah run - Run a command inside of the container.

## SYNOPSIS
**buildah** **run** [*options* [...] --] **containerID** **command**

## DESCRIPTION
Launches a container and runs the specified command in that container using the
container's root filesystem as a root filesystem, using configuration settings
inherited from the container's image or as specified using previous calls to
the *buildah config* command.  To execute *buildah run* within an
interactive shell, specify the --tty option.

## OPTIONS
**--hostname**
Set the hostname inside of the running container.

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime.

**--runtime-flag** *flag*

Adds global flags for the container runtime. To list the supported flags, please
consult the manpages of the selected container runtime (`runc` is the default
runtime, the manpage to consult is `runc(8)`).
Note: Do not pass the leading `--` to the flag. To pass the runc flag `--log-format json`
to buildah run, the option given would be `--runtime-flag log-format=json`.

**--tty**

By default a pseudo-TTY is allocated only when buildah's standard input is
attached to a pseudo-TTY.  Setting the `--tty` option to `true` will cause a
pseudo-TTY to be allocated inside the container connecting the user's "terminal"
with the stdin and stdout stream of the container.  Setting the `--tty` option to
`false` will prevent the pseudo-TTY from being allocated.

**--volume, -v** *source*:*destination*:*flags*

Bind mount a location from the host into the container for its lifetime.

NOTE: End parsing of options with the `--` option, so that other
options can be passed to the command inside of the container.

## EXAMPLE

buildah run containerID -- ps -auxw

buildah run --hostname myhost containerID -- ps -auxw

buildah run --runtime-flag log-format=json containerID /bin/bash

buildah run --runtime-flag debug containerID /bin/bash

buildah run --tty containerID /bin/bash

buildah run --tty=false containerID ls /

## SEE ALSO
buildah(1)
