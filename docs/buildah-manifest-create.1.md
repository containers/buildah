# buildah-manifest-create "16" "August 2022" "buildah"

## NAME

buildah\-manifest\-create - Create a manifest list or image index.

## SYNOPSIS

**buildah manifest create** [options...] *listNameOrIndexName* [*imageName* ...]

## DESCRIPTION

Creates a new manifest list and stores it as an image in local storage using
the specified name.

If additional images are specified, they are added to the newly-created list or
index.

## RETURN VALUE

The randomly-generated image ID of the newly-created list or index.  The image
can be deleted using the *buildah rmi* command.

## OPTIONS

**--all**

If any of the images which should be added to the new list or index are
themselves lists or indexes, add all of their contents.  By default, only one
image from such a list will be added to the newly-created list or index.

**--amend**

If a manifest list named *listNameOrIndexName* already exists, modify the
preexisting list instead of exiting with an error.  The contents of
*listNameOrIndexName* are not modified if no *imageName*s are given.

**--annotation** *annotation=value*

Set an annotation on the newly-created image index.

**--tls-verify** *bool-value*

Require HTTPS and verification of certificates when talking to container registries (defaults to true).  TLS verification cannot be used when talking to an insecure registry.

## EXAMPLE

```
buildah manifest create mylist:v1.11
941c1259e4b85bebf23580a044e4838aa3c1e627528422c9bf9262ff1661fca9
buildah manifest create --amend mylist:v1.11
941c1259e4b85bebf23580a044e4838aa3c1e627528422c9bf9262ff1661fca9
```

```
buildah manifest create mylist:v1.11 docker://fedora
941c1259e4b85bebf23580a044e4838aa3c1e627528422c9bf9262ff1661fca9
```

```
buildah manifest create --all mylist:v1.11 docker://fedora
941c1259e4b85bebf23580a044e4838aa3c1e627528422c9bf9262ff1661fca9
```

## SEE ALSO
buildah(1), buildah-manifest(1), buildah-manifest-add(1), buildah-manifest-remove(1), buildah-manifest-annotate(1), buildah-manifest-inspect(1), buildah-manifest-push(1), buildah-rmi(1)
