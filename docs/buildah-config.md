# buildah-config "1" "March 2017" "buildah"

## NAME
buildah config - Update image configuration settings.

## SYNOPSIS
**buildah** **config** [*options* [...]] **containerID**

## DESCRIPTION
Updates one or more of the settings kept for a container.

## OPTIONS

**--annotation** *annotation*

Adds an image *annotation* (e.g. annotation=*annotation*) to the image manifest
of any images which will be built using the specified container.

**--arch** *architecture*

Specify the target *architecture* for any images which will be built using the
specified container.  By default, if the container was based on an image, that
image's target architecture is kept, otherwise the host's architecture is
recorded.

**--author** *author*

Sets contact information for the *author* for any images which will be built
using the specified container.

**--cmd** *command*

Sets the default *command* to run for containers based on any images which will
be built using the specified container.  When used in combination with an
*entry point*, this specifies the default parameters for the *entry point*.

**--created-by** *created*

Set the description of how the read-write layer *created* (default: "manual
edits") in any images which will be created using the specified container.

**--entrypoint** *entry*

Sets the *entry point* for containers based on any images which will be built
using the specified container.

**--env** *var=value*

Adds a value (e.g. name=*value*) to the environment for containers based on any
images which will be built using the specified container.

**--label** *label*

Adds an image *label* (e.g. label=*value*) to the image configuration of any
images which will be built using the specified container.

**--os** *operating system*

Specify the target *operating system* for any images which will be built using
the specified container.  By default, if the container was based on an image,
its OS is kept, otherwise the host's OS's name is recorded.

**--port** *port*

Specifies a *port* to expose when running containers based on any images which
will be built using the specified container.

**--user** *user*

Specify the *user* as whom containers based on images which will be built using
the specified container should run.  The user can be specified as a user name
or UID, optionally followed by a group name or GID, separated by a colon (':').
If names are used, the container should include entries for those names in its
*/etc/passwd* and */etc/group* files.

**--volume** *volume*

Specifies a location in the directory tree which should be marked as a *volume*
in any images which will be built using the specified container.

**--workingdir** *directory*

Sets the initial working *directory* for containers based on images which will
be built using the specified container.

## EXAMPLE

buildah config --author='Jane Austen' --workingdir='/etc/mycontainers' containerID

## SEE ALSO
buildah(1)
