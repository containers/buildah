% BUILDAH(1) Buildah User Manuals
% Buildah Community
% JUNE 2017
# NAME
buildah-export - Export container's filesystem content as a tar archive

# SYNOPSIS
**buildah export**
[**--help**]
[**-o**|**--output**[=*""*]]
CONTAINER

# DESCRIPTION
**buildah export** exports the full or shortened container ID or container name
to STDOUT and should be redirected to a tar file.

# OPTIONS
**--help**
  Print usage statement

**-o**, **--output**=""
  Write to a file, instead of STDOUT

# EXAMPLES
Export the contents of the container called angry_bell to a tar file
called angry_bell.tar:

    # buildah export angry_bell > angry_bell.tar
    # buildah export --output=angry_bell-latest.tar angry_bell
    # ls -sh angry_bell.tar
    321M angry_bell.tar
    # ls -sh angry_bell-latest.tar
    321M angry_bell-latest.tar

# See also
**buildah-import(1)** to create an empty filesystem image
and import the contents of the tarball into it, then optionally tag it.

# HISTORY
July 2017, Originally copied from docker project docker-export.1.md
