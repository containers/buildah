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
keywords are 'before', 'dangling', 'label', 'readonly' and 'since' .

  Filters:

  **before==TIMESTRING**
    Filter on images created before the given time.Time.

  **dangling=true|false**
    Show dangling images. Dangling images are a file system layer that was used in a previous build of an image and is no longer referenced by any active images. They are denoted with the <none> tag, consume disk space and serve no active purpose.

  **label**
    Filter by images labels key and/or value.

  **readonly=true|false**
     Show only read only images or Read/Write images. The default is to show both.  Read/Only images can be configured by modifying the  "additionalimagestores" in the /etc/containers/storage.conf file.

  **since==TIMESTRING**
    Filter on images created since the given time.Time.

**--format="TEMPLATE"**

Pretty-print images using a Go template.

Valid placeholders for the Go template are listed below:

| **Placeholder** | **Description**                          |
| --------------- | -----------------------------------------|
| .ID             | Image ID                                 |
| .Name           | Image Name                               |
| .Digest         | Image Digest                             |
| .CreatedAt      | Creation date Pretty Formated            |
| .Size           | Image Size                               |
| .CreatedAtRaw   | Creation date in raw format              |
| .ReadOnly       | Indicates if image came from a R/O store |

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

```
# buildah images --format '{{.ID}} {{.CreatedAtRaw}}'
3f53bb00af943dfdf815650be70c0fa7b426e56a66f5e3362b47a129d57d5991 2018-12-20 19:21:30.122610396 -0500 EST
8e09da8f6701d7cde1526d79e3123b0f1109b78d925dfe9f9bac6d59d702a390 2019-01-08 09:22:52.330623532 -0500 EST
```

```
# buildah images --format '{{.ID}} {{.Name}} {{.Digest}} {{.CreatedAt}} {{.Size}} {{.CreatedAtRaw}}'
3f53bb00af943dfdf815650be70c0fa7b426e56a66f5e3362b47a129d57d5991 docker.io/library/alpine sha256:3d2e482b82608d153a374df3357c0291589a61cc194ec4a9ca2381073a17f58e Dec 20, 2018 19:21 4.67 MB 2018-12-20 19:21:30.122610396 -0500 EST
8e09da8f6701d7cde1526d79e3123b0f1109b78d925dfe9f9bac6d59d702a390 <none> sha256:894532ec56e0205ce68ca7230b00c18aa3c8ee39fcdb310615c60e813057229c Jan 8, 2019 09:22 4.67 MB 2019-01-08 09:22:52.330623532 -0500 EST
```
## SEE ALSO
buildah(1), containers-storage.conf(5)
