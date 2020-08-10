# buildah-add "1" "March 2017" "buildah"

## NAME
buildah\-add - Add the contents of a file, URL, or a directory to a container.

## SYNOPSIS
**buildah add** [*options*] *container* *src* [[*src* ...] *dest*]

## DESCRIPTION
Adds the contents of a file, URL, or a directory to a container's working
directory or a specified location in the container.  If a local source file
appears to be an archive, its contents are extracted and added instead of the
archive file itself.  If a local directory is specified as a source, its
*contents* are copied to the destination.

## OPTIONS

**--add-history**

Add an entry to the history which will note the digest of the added content.
Defaults to false.

Note: You can also override the default value of --add-history by setting the
BUILDAH\_HISTORY environment variable. `export BUILDAH_HISTORY=true`

**--chown** *owner*:*group*

Sets the user and group ownership of the destination content.

**--quiet**

Refrain from printing a digest of the added content.

## EXAMPLE

buildah add containerID '/myapp/app.conf' '/myapp/app.conf'

buildah add --chown myuser:mygroup containerID '/myapp/app.conf' '/myapp/app.conf'

buildah add containerID '/home/myuser/myproject.go'

buildah add containerID '/home/myuser/myfiles.tar' '/tmp'

buildah add containerID '/tmp/workingdir' '/tmp/workingdir'

buildah add containerID 'https://github.com/containers/buildah/blob/master/README.md' '/tmp'

buildah add containerID 'passwd' 'certs.d' /etc

## FILES

### `.dockerignore`

If the file .dockerignore exists in the context directory, `buildah copy` reads
its contents. Buildah uses the content to exclude files and directories from
the context directory, when copying content into the image.

Users can specify a series of Unix shell globals in a .dockerignore file to
identify files/directories to exclude.

Buildah supports a special wildcard string `**` which matches any number of
directories (including zero). For example, **/*.go will exclude all files that
end with .go that are found in all directories.

Example .dockerignore file:

```
# here are files we want to exclude
*/*.c
**/output*
src
```

`*/*.c`
Excludes files and directories whose names ends with .c in any top level subdirectory. For example, the source file include/rootless.c.

`**/output*`
Excludes files and directories starting with `output` from any directory.

`src`
Excludes files named src and the directory src as well as any content in it.

Lines starting with ! (exclamation mark) can be used to make exceptions to
exclusions. The following is an example .dockerignore file that uses this
mechanism:
```
*.doc
!Help.doc
```

Exclude all doc files except Help.doc from the image.

This functionality is compatible with the handling of .dockerignore files described here:

https://docs.docker.com/engine/reference/builder/#dockerignore-file

## SEE ALSO
buildah(1)
