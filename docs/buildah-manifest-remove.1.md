# buildah-manifest-remove "1" "September 2019" "buildah"

## NAME

buildah\-manifest\-remove - Remove an image from a manifest list or image index.

## SYNOPSIS

**buildah manifest remove** *listNameOrIndexName* *imageManifestDigest*

## DESCRIPTION

Removes the image with the specified digest from the specified manifest list or image index.

## RETURN VALUE

The list image's ID and the digest of the removed image's manifest.

## EXAMPLE

```
buildah manifest remove mylist:v1.11 sha256:f81f09918379d5442d20dff82a298f29698197035e737f76e511d5af422cabd7
506d8f4bb54931ea03a7e70173a0ed6302e3fb92dfadb3955ba5c17812e95c51: sha256:f81f09918379d5442d20dff82a298f29698197035e737f76e511d5af422cabd7
```

## SEE ALSO
buildah(1), buildah-manifest(1), buildah-manifest-create(1), buildah-manifest-add(1), buildah-manifest-annotate(1), buildah-manifest-inspect(1), buildah-manifest-push(1), buildah-rmi(1)
