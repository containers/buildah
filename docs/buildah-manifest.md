# buildah-manifest "1" "September 2019" "buildah"

## NAME
buildah-manifest - Create and manipulate manifest lists and image indexes.

## SYNOPSIS
buildah manifest COMMAND [OPTIONS] [ARG...]

## DESCRIPTION
The `buildah manifest` command provides subcommands which can be used to:

    * Create a working Docker manifest list or OCI image index.
    * Add an entry to a manifest list or image index for a specified image.
    * Add or update information about an entry in a manifest list or image index.
    * Delete a working container or an image.
    * Push a manifest list or image index to a registry or other location.

## SUBCOMMANDS

| Command  | Man Page                                                     | Description                                                                 |
| -------  | ------------------------------------------------------------ | --------------------------------------------------------------------------- |
| add      | [buildah-manifest-add(1)](buildah-manifest-add.md)           | Add an image to a manifest list or image index.                             |
| annotate | [buildah-manifest-annotate(1)](buildah-manifest-annotate.md) | Add or update information about an image in a manifest list or image index. |
| create   | [buildah-manifest-create(1)](buildah-manifest-create.md)     | Create a manifest list or image index.                                      |
| inspect  | [buildah-manifest-inspect(1)](buildah-manifest-inspect.md)   | Display the contents of a manifest list or image index.                     |
| push     | [buildah-manifest-push(1)](buildah-manifest-push.md)         | Push a manifest list or image index to a registry or other location.        |
| remove   | [buildah-manifest-remove(1)](buildah-manifest-remove.md)     | Remove an image from a manifest list or image index.                        |
| rm       | [buildah-manifest-rm(1)](buildah-manifest-rm.md)             | Remove manifest list from local storage.                                    |

## SEE ALSO
buildah(1), buildah-manifest-create(1), buildah-manifest-add(1), buildah-manifest-remove(1), buildah-manifest-annotate(1), buildah-manifest-inspect(1), buildah-manifest-push(1), buildah-manifest-rm(1)
