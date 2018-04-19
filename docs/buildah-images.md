# buildah-images "1" "March 2017" "buildah"

## NAME
buildah images - List images in local storage.

## SYNOPSIS
**buildah** **images** [*options* [...]]

## DESCRIPTION
Displays locally stored images, their names, sizes, created date and their IDs.
The created date is displayed in the time locale of the local machine.

## OPTIONS

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

**no-trunc**

Do not truncate output.

**--quiet, -q**

Displays only the image IDs.

## EXAMPLE

buildah images

buildah images --json

buildah images --quiet

buildah images -q --noheading --notruncate

buildah images --filter dangling=true

buildah images --format "ImageID: {{.ID}}"

## SEE ALSO
buildah(1)
