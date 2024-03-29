# buildah-manifest-annotate "1" "September 2019" "buildah"

## NAME

buildah\-manifest\-annotate - Add and update information about an image or artifact to a manifest list or image index.

## SYNOPSIS

**buildah manifest annotate** [options...] *listNameOrIndexName* *imageManifestDigestOrImageOrArtifactName*

## DESCRIPTION

Adds or updates information about an image or artifact included in a manifest list or image index.

## RETURN VALUE

The list image's ID and the digest of the image's manifest.

## OPTIONS

**--annotation** *annotation=value*

Set an annotation on the entry for the specified image or artifact.  If
**--index** is also specified, sets the annotation on the entire image index.

**--arch**

Override the architecture which the list or index records as a requirement for
the image.  This is usually automatically retrieved from the image's
configuration information, so it is rarely necessary to use this option.

**--features**

Specify the features list which the list or index records as requirements for
the image.  This option is rarely used.

**--index**

Treats arguments to the **--annotation** option as annotation values to be set
on the image index itself rather than on an entry in the image index.  Implied
for **--subject**.

**--os**

Override the OS which the list or index records as a requirement for the image.
This is usually automatically retrieved from the image's configuration
information, so it is rarely necessary to use this option.

**--os-features**

Specify the OS features list which the list or index records as requirements
for the image.  This option is rarely used.

**--os-version**

Specify the OS version which the list or index records as a requirement for the
image.  This option is rarely used.

**--subject** *imageName*

Set the *subject* field in the image index to mark the image index as being
associated with the specified image in some way.  An image index can only be
associated with, at most, one subject.

**--variant**

Specify the variant which the list or index records for the image.  This option
is typically used to distinguish between multiple entries which share the same
architecture value, but which expect different versions of its instruction set.

## EXAMPLE

```
buildah manifest annotate --arch arm64 --variant v8 mylist:v1.11 sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b
506d8f4bb54931ea03a7e70173a0ed6302e3fb92dfadb3955ba5c17812e95c51: sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b
```

```
buildah manifest annotate --index --annotation food=yummy mylist:v1.11
506d8f4bb54931ea03a7e70173a0ed6302e3fb92dfadb3955ba5c17812e95c51: sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b
```

## SEE ALSO
buildah(1), buildah-manifest(1), buildah-manifest-create(1), buildah-manifest-add(1), buildah-manifest-remove(1), buildah-manifest-inspect(1), buildah-manifest-push(1), buildah-rmi(1)
