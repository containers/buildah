## buildah-import "1" "July 2017" "buildah"
# NAME
buildah-import - Create an empty filesystem image and import the contents of the tar archive (.tar, .tar.gz, .tgz, .bzip, .tar.xz, .txz) into it, then optionally tag it.

# SYNOPSIS
**buildah** **import** [*options* [...]] file|URL|**-**[REPOSITORY[:TAG]]

# OPTIONS
**-c**, **--change**=[]
   Apply specified Dockerfile instructions while importing the image
   Supported Dockerfile instructions: `CMD`|`ENTRYPOINT`|`ENV`|`EXPOSE`|`ONBUILD`|`USER`|`VOLUME`|`WORKDIR`

**--help**
  Print usage statement

**-m**, **--message**=""
   Set commit message for imported image

**--signature-policy** *signaturepolicy*

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

# DESCRIPTION
Create a new filesystem image from the contents of a tar archive (`.tar`,
`.tar.gz`, `.tgz`, `.bzip`, `.tar.xz`, `.txz`) into it, then optionally tag it.


# EXAMPLES

## Import from a remote location

    # buildah import http://example.com/exampleimage.tgz example/imagerepo

## Import from a local file

Import to buildah via pipe and stdin:

    # cat exampleimage.tgz | buildah import - example/imagelocal

Import with a commit message.

    # cat exampleimage.tgz | buildah import --message "New image imported from tar archive" - exampleimagelocal:new

Import to a Buildah image from a local file.

    # buildah import /path/to/exampleimage.tgz


## Import from a local file and tag

Import to buildah via pipe and stdin:

    # cat exampleimageV2.tgz | buildah import - example/imagelocal:V-2.0

## Import from a local directory

    # tar -c . | buildah import - exampleimagedir

## Apply specified Dockerfile instructions while importing the image
This example sets the buildah image ENV variable DEBUG to true by default.

    # tar -c . | buildah import -c="ENV DEBUG true" - exampleimagedir

# See also
**buildah-export(1)** to export the contents of a filesystem as a tar archive to STDOUT.

# HISTORY
July 2017, Originally copied from docker project docker-import.1.md
