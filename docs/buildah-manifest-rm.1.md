# buildah-manifest-rm "1" "April 2021" "buildah"

## NAME
buildah\-manifest\-rm - Removes one or more manifest lists.

## SYNOPSIS
**buildah manifest rm** [options...] [*listNameOrIndexName* ...]

## DESCRIPTION
Removes one or more locally stored manifest lists.

## OPTIONS

**--tls-details** *path*

Path to a `containers-tls-details.yaml(5)` file.

If not set, defaults to a reasonable default that may change over time (depending on system's global policy,
version of the program, version of the Go language, and the like).

Users should generally not use this option unless they have a process to ensure that the configuration will be kept up to date.

## EXAMPLE

buildah manifest rm <list>

buildah manifest-rm listID1 listID2

**storage.conf** (`/etc/containers/storage.conf`)

storage.conf is the storage configuration file for all tools using containers/storage

The storage configuration file specifies all of the available container storage options for tools using shared container storage.

## SEE ALSO
buildah(1), containers-storage.conf(5), buildah-manifest(1)
