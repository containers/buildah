# buildah-copy "1" "April 2021" "buildah"

## NAME
buildah\-copy - Copies the contents of a file, URL, or directory into a container's working directory.

## SYNOPSIS
**buildah copy** *container* *src* [[*src* ...] *dest*]

## DESCRIPTION
Copies the contents of a file, URL, or a directory to a container's working
directory or a specified location in the container.  If a local directory is
specified as a source, its *contents* are copied to the destination.

## OPTIONS

**--add-history**

Add an entry to the history which will note the digest of the added content.
Defaults to false.

Note: You can also override the default value of --add-history by setting the
BUILDAH\_HISTORY environment variable. `export BUILDAH_HISTORY=true`

**--checksum** *checksum*

Checksum the source content. The value of *checksum* must be a standard
container digest string. Only supported for HTTP sources.

**--chmod** *permissions*

Sets the access permissions of the destination content.  Accepts the numerical
format.  If `--from` is not used, defaults to `0755`.

**--chown** *owner*:*group*

Sets the user and group ownership of the destination content.  If `--from` is
not used, defaults to `0:0`.

**--contextdir** *directory*

Build context directory. Specifying a context directory causes Buildah to
chroot into the context directory. This means copying files pointed at
by symbolic links outside of the chroot will fail.

**--from** *containerOrImage*

Use the root directory of the specified working container or image as the root
directory when resolving absolute source paths and the path of the context
directory.  If an image needs to be pulled, options recognized by `buildah pull`
can be used.  If `--chown` or `--chmod` are not used, permissions and ownership
is preserved.

**--ignorefile** *file*

Path to an alternative .containerignore (.dockerignore) file. Requires \-\-contextdir be specified.

**--quiet**, **-q**

Refrain from printing a digest of the copied content.

**--retry** *attempts*

Number of times to retry in case of failure when performing pull of images from registry.

Defaults to `3`.

**--retry-delay** *duration*

Duration of delay between retry attempts in case of failure when performing pull of images from registry.

Defaults to `2s`.

## EXAMPLE

buildah copy containerID '/myapp/app.conf' '/myapp/app.conf'

buildah copy --chown myuser:mygroup containerID '/myapp/app.conf' '/myapp/app.conf'

buildah copy --chmod 660 containerID '/myapp/app.conf' '/myapp/app.conf'

buildah copy containerID '/home/myuser/myproject.go'

buildah copy containerID '/home/myuser/myfiles.tar' '/tmp'

buildah copy containerID '/tmp/workingdir' '/tmp/workingdir'

buildah copy containerID 'https://github.com/containers/buildah' '/tmp'

buildah copy containerID 'passwd' 'certs.d' /etc

## FILES

### .containerignore/.dockerignore

If the .containerignore/.dockerignore file exists in the context directory,
`buildah copy` reads its contents. If both exist, then .containerignore is used.

When the `--ignorefile` option is specified Buildah reads it and
uses it to decide which content to exclude when copying content into the
working container.

Users can specify a series of Unix shell glob patterns in an ignore file to
identify files/directories to exclude.

Buildah supports a special wildcard string `**` which matches any number of
directories (including zero). For example, **/*.go will exclude all files that
end with .go that are found in all directories.

Example .containerignore/.dockerignore file:

```
# here are files we want to exclude
*/*.c
**/output*
src
```

`*/*.c`
Excludes files and directories whose names end with .c in any top level subdirectory. For example, the source file include/rootless.c.

`**/output*`
Excludes files and directories starting with `output` from any directory.

`src`
Excludes files named src and the directory src as well as any content in it.

Lines starting with ! (exclamation mark) can be used to make exceptions to
exclusions. The following is an example .containerignore/.dockerignore file that uses this
mechanism:
```
*.doc
!Help.doc
```

Exclude all doc files except Help.doc when copying content into the container.

This functionality is compatible with the handling of .containerignore files described here:

https://github.com/containers/common/blob/main/docs/containerignore.5.md

## SEE ALSO
buildah(1), containerignore(5)
