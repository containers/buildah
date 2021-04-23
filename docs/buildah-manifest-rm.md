# buildah-manifest-rm "1" "April 2021" "buildah"

## NAME
buildah\-manifest\-rm - Removes one or more manifest lists.

## SYNOPSIS
**buildah manifest rm** [*listNameOrIndexName* ...]

## DESCRIPTION
Removes one or more locally stored manifest lists.

## EXAMPLE

buildah manifest rm <list>

buildah manifest-rm listID1 listID2

**storage.conf** (`/etc/containers/storage.conf`)

storage.conf is the storage configuration file for all tools using containers/storage

The storage configuration file specifies all of the available container storage options for tools using shared container storage.

## SEE ALSO
buildah(1), containers-storage.conf(5), buildah-manifest(1)
