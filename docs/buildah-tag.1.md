# buildah-tag "1" "May 2017" "buildah"

## NAME
buildah\-tag - Add additional names to local images.

## SYNOPSIS
**buildah tag** *name* *new-name* ...

## DESCRIPTION
Adds additional names to locally-stored images.

## OPTIONS

**--tls-details** *path*

Path to a `containers-tls-details.yaml(5)` file, affecting TLS behavior throughout the program.

If not set, defaults to a reasonable default that may change over time (depending on system’s global policy,
version of the program, version of the Go language, and the like).

Users should generally not use this option unless they have a process to ensure that the configuration will be kept up to date.

## EXAMPLE

buildah tag imageName firstNewName

buildah tag imageName firstNewName SecondNewName

## SEE ALSO
buildah(1)
