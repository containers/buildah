## buildah-inspect "1" "May 2017" "buildah"

## NAME
buildah inspect - Display information about a working container.

## SYNOPSIS
**buildah** **inspect** [*options* [...] --] **ID**

## DESCRIPTION
Prints information about a working container's configuration, or the initial
configuration for a container which would be created for an image.

## OPTIONS

**--format** *template*

Use *template* as a Go template when formatting the output.

Users of this option should be familiar with the [*text/template*
package](https://golang.org/pkg/text/template/) in the Go standard library, and
of internals of buildah's implementation.

**--type** *container* | *image*

Specify whether the ID is that of a container or an image.

## EXAMPLE

buildah inspect containerID
buildah inspect --type container containerID
buildah inspect --type image imageID

## SEE ALSO
buildah(1)
