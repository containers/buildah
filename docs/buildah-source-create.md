# buildah-source-create "1" "March 2021" "buildah"

## NAME
buildah\-source\-create - Create and initialize a source image

## SYNOPSIS
**buildah source create** [*options*] *path*

## DESCRIPTION
Create and initialize a source image.  A source image is an OCI artifact; an
OCI image with a custom config media type.

Note that the buildah-source command and all its subcommands are experimental
and may be subject to future changes

## OPTIONS
**--author** *author*

Set the author of the source image mentioned in the config.  By default, no author is set.

**--time-stamp** *bool-value*

Set the created time stamp in the image config.  By default, the time stamp is set.
