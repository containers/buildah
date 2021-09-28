# buildah-manifest-inspect "1" "September 2019" "buildah"

## NAME

buildah\-manifest\-inspect - Display a manifest list or image index.

## SYNOPSIS

**buildah manifest inspect** *listNameOrIndexName*

## DESCRIPTION

Displays the manifest list or image index stored using the specified image name.

## RETURN VALUE

A formatted JSON representation of the manifest list or image index.

## EXAMPLE

```
buildah manifest inspect mylist:v1.11
```

## SEE ALSO
buildah(1), buildah-manifest(1), buildah-manifest-create(1), buildah-manifest-add(1), buildah-manifest-remove(1), buildah-manifest-annotate(1), buildah-manifest-push(1), buildah-rmi(1)
