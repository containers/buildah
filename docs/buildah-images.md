# buildah-images "1" "March 2017" "buildah"

## NAME
buildah\-images - List images in local storage.

## SYNOPSIS
**buildah images** [*options*] [*image*]

## DESCRIPTION
Displays locally stored images, their names, sizes, created date and their IDs.
The created date is displayed in the time locale of the local machine.

## OPTIONS

**--all, -a**

Show all images, including intermediate images from a build.

**--digests**

Show the image digests.

**--filter, -f=[]**

Filter output based on conditions provided (default []).  Valid
keywords are 'dangling', 'label', 'before' and 'since'.

**--format="TEMPLATE"**

Pretty-print images using a Go template.

**--json**

Display the output in JSON format.

**--noheading, -n**

Omit the table headings from the listing of images.

**--no-trunc, --notruncate**

Do not truncate output.

**--quiet, -q**

Displays only the image IDs.

## EXAMPLE

buildah images

buildah images fedora:latest

buildah images --json

buildah images --quiet

buildah images -q --noheading --notruncate

buildah images --quiet fedora:latest

buildah images --filter dangling=true

buildah images --format "ImageID: {{.ID}}"

```
# buildah images
IMAGE ID             IMAGE NAME                                               CREATED AT             SIZE
3fd9065eaf02         docker.io/library/alpine:latest                          Jan 9, 2018 16:10      4.41 MB
c0cfe75da054         localhost/test:latest                                    Jun 13, 2018 15:52     4.42 MB
```

```
# buildah images -a
IMAGE ID             IMAGE NAME                                               CREATED AT             SIZE
3fd9065eaf02         docker.io/library/alpine:latest                          Jan 9, 2018 16:10      4.41 MB
12515a2658dc         <none>                                                   Jun 13, 2018 15:52     4.41 MB
fcc3ddd28930         <none>                                                   Jun 13, 2018 15:52     4.41 MB
8c6e16890c2b         <none>                                                   Jun 13, 2018 15:52     4.42 MB
c0cfe75da054         localhost/test:latest                                    Jun 13, 2018 15:52     4.42 MB
```

## SEE ALSO
buildah(1)
