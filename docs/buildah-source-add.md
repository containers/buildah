# buildah-source-add "1" "March 2021" "buildah"

## NAME
buildah\-source\-add - Add a source artifact to a source image

## SYNOPSIS
**buildah source add** [*options*] *path* *artifact*

## DESCRIPTION
Add add a source artifact to a source image.  The artifact will be added as a
gzip-compressed tar ball.  Add attempts to auto-tar and auto-compress only if
necessary.

Note that the buildah-source command and all its subcommands are experimental
and may be subject to future changes

## OPTIONS

**--annotation** *key=value*

Add an annotation to the layer descriptor in the source-image manifest.  The input format is `key=value`.
