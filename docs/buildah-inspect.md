# buildah-inspect "1" "May 2017" "buildah"

## NAME
buildah\-inspect - Display information about working containers or images.

## SYNOPSIS
**buildah inspect** [*options*] [**--**] *object*

## DESCRIPTION
Prints the low-level information on Buildah object(s) (e.g. container, images) identified by name or ID. By default, this will render all results in a
JSON array. If the container and image have the same name, this will return container JSON for unspecified type. If a format is specified, 
the given template will be executed for each result.

## OPTIONS

**--format** *template*

Use *template* as a Go template when formatting the output.

Users of this option should be familiar with the [*text/template*
package](https://golang.org/pkg/text/template/) in the Go standard library, and
of internals of Buildah's implementation.

**--type** **container** | **image**

Specify whether *object* is a container or an image.

## EXAMPLE

buildah inspect containerID

buildah inspect --type container containerID

buildah inspect --type image imageID

buildah inspect --format '{{.OCIv1.Config.Env}}' alpine

## SEE ALSO
buildah(1)
