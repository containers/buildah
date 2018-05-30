# buildah-run "1" "March 2017" "buildah"

## NAME
buildah\-run - Run a command inside of the container.

## SYNOPSIS
**buildah** **run** [*options* [...] --] **containerID** **command**

## DESCRIPTION
Launches a container and runs the specified command in that container using the
container's root filesystem as a root filesystem, using configuration settings
inherited from the container's image or as specified using previous calls to
the *buildah config* command.  To execute *buildah run* within an
interactive shell, specify the --tty option.

## OPTIONS
**--cni-config-dir**=*directory*

Location of CNI configuration files which will dictate which plugins will be
used to configure network interfaces and routing inside the running container,
if the container will be run in its own network namespace, and networking is
not disabled.

**--cni-plugin-path**=*directory[:directory[:directory[...]]]*

List of directories in which the CNI plugins which will be used for configuring
network namespaces can be found.

**--hostname**
Set the hostname inside of the running container.

**--ipc** *how*

Sets the configuration for the IPC namespaces for the container.
The configured value can be "" (the empty string) or "container" to indicate
that a new IPC namespace should be created, or it can be "host" to indicate
that the IPC namespace in which `buildah` itself is being run should be reused,
or it can be the path to an IPC namespace which is already in use by another
process.

**--net** *how*
**--network** *how*

Sets the configuration for the network namespace for the container.
The configured value can be "" (the empty string) or "container" to indicate
that a new network namespace should be created, or it can be "host" to indicate
that the network namespace in which `buildah` itself is being run should be
reused, or it can be the path to a network namespace which is already in use by
another process.

**--pid** *how*

Sets the configuration for the PID namespace for the container.
The configured value can be "" (the empty string) or "container" to indicate
that a new PID namespace should be created, or it can be "host" to indicate
that the PID namespace in which `buildah` itself is being run should be reused,
or it can be the path to a PID namespace which is already in use by another
process.

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime.

**--runtime-flag** *flag*

Adds global flags for the container runtime. To list the supported flags, please
consult the manpages of the selected container runtime (`runc` is the default
runtime, the manpage to consult is `runc(8)`).
Note: Do not pass the leading `--` to the flag. To pass the runc flag `--log-format json`
to buildah run, the option given would be `--runtime-flag log-format=json`.

**-t**, **--tty**, **--terminal**

By default a pseudo-TTY is allocated only when buildah's standard input is
attached to a pseudo-TTY.  Setting the `--tty` option to `true` will cause a
pseudo-TTY to be allocated inside the container connecting the user's "terminal"
with the stdin and stdout stream of the container.  Setting the `--tty` option to
`false` will prevent the pseudo-TTY from being allocated.

**--user** *user*[:*group*]

Set the *user* to be used for running the command in the container.
The user can be specified as a user name
or UID, optionally followed by a group name or GID, separated by a colon (':').
If names are used, the container should include entries for those names in its
*/etc/passwd* and */etc/group* files.

**--uts** *how*

Sets the configuration for the UTS namespace for the container.
The configured value can be "" (the empty string) or "container" to indicate
that a new UTS namespace should be created, or it can be "host" to indicate
that the UTS namespace in which `buildah` itself is being run should be reused,
or it can be the path to a UTS namespace which is already in use by another
process.

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
buildah(1), namespaces(7), pid\_namespaces(7)
