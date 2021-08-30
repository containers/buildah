# buildah-source "1" "March 2021" "buildah"

## NAME
buildah\-source - Create, push, pull and manage source images and associated source artifacts

## SYNOPSIS
**buildah source** *subcommand*

## DESCRIPTION
Create, push, pull and manage source images and associated source artifacts.  A
source image contains all source artifacts an ordinary OCI image has been built
with.  Those artifacts can be any kind of source artifact, such as source RPMs,
an entire source tree or text files.

Note that the buildah-source command and all its subcommands are experimental
and may be subject to future changes.

## COMMANDS
| Command  | Man Page                                             | Description                                                |
| -------- | ---------------------------------------------------- | ---------------------------------------------------------- |
| add      | [buildah-source-add(1)](buildah-source-add.md)       | Add a source artifact to a source image.                   |
| create   | [buildah-source-create(1)](buildah-source-create.md) | Create and initialize a source image.                      |
| pull     | [buildah-source-pull(1)](buildah-source-pull.md)     | Pull a source image from a registry to a specified path.   |
| push     | [buildah-source-push(1)](buildah-source-push.md)     | Push a source image from a specified path to a registry.   |
