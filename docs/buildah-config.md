# buildah-config "1" "March 2017" "buildah"

## NAME
buildah\-config - Update image configuration settings.

## SYNOPSIS
**buildah config** [*options*] *container*

## DESCRIPTION
Updates one or more of the settings kept for a container.

## OPTIONS

**--add-history**

Add an entry to the image's history which will note changes to the settings for
**--cmd**, **--entrypoint**, **--env**, **--healthcheck**, **--label**,
**--onbuild**, **--port**, **--shell**, **--stop-signal**, **--user**,
**--volume**, and **--workingdir**.
Defaults to false.

Note: You can also override the default value of --add-history by setting the
BUILDAH\_HISTORY environment variable. `export BUILDAH_HISTORY=true`

**--annotation** *annotation*=*annotation*

Add an image *annotation* (e.g. annotation=*annotation*) to the image manifest
of any images which will be built using the specified container. Can be used multiple times.
If *annotation* has a trailing `-`, then the *annotation* is removed from the config.

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

**--env** *env=value*

Add a value (e.g. env=*value*) to the environment for containers based on any
images which will be built using the specified container. Can be used multiple times.
If *env* has a trailing `-`, then the *env* is removed from the config.

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

Specify how much time can elapse after a container has started before a failure
to run the command specified using the *--healthcheck* option should be treated
as an indication that the container is failing.  During this time period,
failures will be attributed to the container not yet having fully started, and
will not be counted as errors.  After the command succeeds, or the time period
has elapsed, failures will be counted as errors.

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

**--label** *label*=*value*

Add an image *label* (e.g. label=*value*) to the image configuration of any
images which will be built using the specified container. Can be used multiple times.
If *label* has a trailing `-`, then the *label* is removed from the config.

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

Add a location in the directory tree which should be marked as a *volume* in any images which will be built using the specified container. Can be used multiple times. If *volume* has a trailing `-`, and is already set, then the *volume* is removed from the config.

**--workingdir** *directory*

Set the initial working *directory* for containers based on images which will
be built using the specified container.

## EXAMPLE

buildah config --author='Jane Austen' --workingdir='/etc/mycontainers' containerID

buildah config --entrypoint /entrypoint.sh containerID

buildah config --entrypoint '[ "/entrypoint.sh", "dev" ]' containerID

buildah config --env foo=bar --env PATH=$PATH containerID

buildah config --env foo- containerID

buildah config --label Name=Mycontainer --label  Version=1.0 containerID

buildah config --label Name- containerID

buildah config --annotation note=myNote containerID

buildah config --annotation note-

buildah config --volume /usr/myvol containerID

buildah config --volume /usr/myvol- containerID


## SEE ALSO
buildah(1)
