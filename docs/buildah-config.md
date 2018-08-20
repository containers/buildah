# buildah-config "1" "March 2017" "buildah"

## NAME
buildah\-config - Update image configuration settings.

## SYNOPSIS
**buildah config** [*options*] *container*

## DESCRIPTION
Updates one or more of the settings kept for a container.

## OPTIONS

**--annotation** *annotation*

Add an image *annotation* (e.g. annotation=*annotation*) to the image manifest
of any images which will be built using the specified container. Can be used multiple times.

**--arch** *architecture*

Set the target *architecture* for any images which will be built using the
specified container.  By default, if the container was based on an image, that
image's target architecture is kept, otherwise the host's architecture is
recorded.

**--author** *author*

Set contact information for the *author* for any images which will be built
using the specified container.

**--cmd** *command*

Set the default *command* to run for containers based on any images which will
be built using the specified container.  When used in combination with an
*entry point*, this specifies the default parameters for the *entry point*.

**--comment** *comment*

Set the image-level comment for any images which will be built using the
specified container.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--created-by** *created*

Set the description of how the topmost layer was *created* for any images which
will be created using the specified container.

**--domainname** *domain*

Set the domainname to set when running containers based on any images built
using the specified container.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--entrypoint** *"command"* | *'["command", "arg1", ...]'*

Set the *entry point* for containers based on any images which will be built
using the specified container. buildah supports two formats for entrypoint.  It
can be specified as a simple string, or as a array of commands.

Note: When the entrypoint is specified as a string, container runtimes will
ignore the `cmd` value of the container image.  However if you use the array
form, then the cmd will be appended onto the end of the entrypoint cmd and be
executed together.

**--env** *var=value*

Add a value (e.g. name=*value*) to the environment for containers based on any
images which will be built using the specified container. Can be used multiple times.

**--healthcheck** *command*

Specify a command which should be run to check if a container is running correctly.

Values can be *NONE*, "*CMD* ..." (run the specified command directly), or
"*CMD-SHELL* ..." (run the specified command using the system's shell), or the
empty value (remove a previously-set value and related settings).

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--healthcheck-interval** *interval*

Specify how often the command specified using the *--healthcheck* option should
be run.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--healthcheck-retries** *count*

Specify how many times the command specified using the *--healthcheck* option
can fail before the container is considered to be unhealthy.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--healthcheck-start-period** *interval*

Specify how long to wait after starting a container before running the command
specified using the *--healthcheck* option.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--healthcheck-timeout** *interval*

Specify how long to wait after starting the command specified using the
*--healthcheck* option to wait for the command to return its exit status.  If
the command has not returned within this time, it should be considered to have
failed.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--history-comment** *comment*

Sets a comment on the topmost layer in any images which will be created
using the specified container.

**--hostname** *host*

Set the hostname to set when running containers based on any images built using
the specified container.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--label** *label*

Add an image *label* (e.g. label=*value*) to the image configuration of any
images which will be built using the specified container. Can be used multiple times.

**--onbuild** *onbuild command*

Add an ONBUILD command to the image.  ONBUILD commands are automatically run
when images are built based on the image you are creating.

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--os** *operating system*

Set the target *operating system* for any images which will be built using
the specified container.  By default, if the container was based on an image,
its OS is kept, otherwise the host's OS's name is recorded.

**--port** *port*

Add a *port* to expose when running containers based on any images which
will be built using the specified container. Can be used multiple times.

**--shell** *shell*

Set the default *shell* to run inside of the container image.
The shell instruction allows the default shell used for the shell form of commands to be overridden. The default shell for Linux containers is "/bin/sh -c".

Note: this setting is not present in the OCIv1 image format, so it is discarded when writing images using OCIv1 formats.

**--stop-signal** *signal*

Set default *stop signal* for container. This signal will be sent when container is stopped, default is SIGINT.

**--user** *user*[:*group*]

Set the default *user* to be used when running containers based on this image.
The user can be specified as a user name
or UID, optionally followed by a group name or GID, separated by a colon (':').
If names are used, the container should include entries for those names in its
*/etc/passwd* and */etc/group* files.

**--volume** *volume*

Add a location in the directory tree which should be marked as a *volume* in any images which will be built using the specified container. Can be used multiple times.

**--workingdir** *directory*

Set the initial working *directory* for containers based on images which will
be built using the specified container.

## EXAMPLE

buildah config --author='Jane Austen' --workingdir='/etc/mycontainers' containerID

buildah config --entrypoint /entrypoint.sh containerID

buildah config --entrypoint '[ "/entrypoint.sh", "dev" ]' containerID

buildah config --env foo=bar PATH=$PATH containerID

buildah config --label Name=Mycontainer --label  Version=1.0 containerID

## SEE ALSO
buildah(1)
