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
IMAGE NAME                                               IMAGE TAG            IMAGE ID             CREATED AT             SIZE
docker.io/library/alpine                                 latest               3fd9065eaf02         Jan 9, 2018 16:10      4.41 MB
localhost/test                                           latest               c0cfe75da054         Jun 13, 2018 15:52     4.42 MB
```

```
# buildah images -a
IMAGE NAME                                               IMAGE TAG            IMAGE ID             CREATED AT             SIZE
docker.io/library/alpine                                 latest               3fd9065eaf02         Jan 9, 2018 16:10      4.41 MB
<none>                                                   <none>               12515a2658dc         Jun 13, 2018 15:52     4.41 MB
<none>                                                   <none>               fcc3ddd28930         Jun 13, 2018 15:52     4.41 MB
<none>                                                   <none>               8c6e16890c2b         Jun 13, 2018 15:52     4.42 MB
localhost/test                                           latest               c0cfe75da054         Jun 13, 2018 15:52     4.42 MB
```

## SEE ALSO
buildah(1)
