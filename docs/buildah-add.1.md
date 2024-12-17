# buildah-add "1" "April 2021" "buildah"

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

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) when connecting to
registries for pulling images named with the **--from** flag, and when
connecting to HTTPS servers when fetching sources from locations specified with
HTTPS URLs.  The default certificates directory is _/etc/containers/certs.d_.

**--checksum** *checksum*

Checksum the source content. The value of *checksum* must be a standard
container digest string. Only supported for HTTP sources.

**--chmod** *permissions*

Sets the access permissions of the destination content. Accepts the numerical format.

**--chown** *owner*:*group*

Sets the user and group ownership of the destination content.

**--contextdir** *directory*

Build context directory. Specifying a context directory causes Buildah to
chroot into that context directory. This means copying files pointed at
by symbolic links outside of the chroot will fail.

**--exclude** *pattern*

Exclude copying files matching the specified pattern. Option can be specified
multiple times. See containerignore(5) for supported formats.

**--from** *containerOrImage*

Use the root directory of the specified working container or image as the root
directory when resolving absolute source paths and the path of the context
directory.  If an image needs to be pulled, options recognized by `buildah pull`
can be used.

**--ignorefile** *file*

Path to an alternative .containerignore (.dockerignore) file. Requires \-\-contextdir be specified.

**--quiet**, **-q**

Refrain from printing a digest of the added content.

**--retry** *attempts*

Number of times to retry in case of failure when pulling images from registries
or retrieving content from HTTPS URLs.

Defaults to `3`.

**--retry-delay** *duration*

Duration of delay between retry attempts in case of failure when pulling images
from registries or retrieving content from HTTPS URLs.

Defaults to `2s`.

**--tls-verify** *bool-value*

Require verification of certificates when retrieving sources from HTTPS
locations, or when pulling images referred to with the **--from*** flag
(defaults to true).  TLS verification cannot be used when talking to an
insecure registry.

## EXAMPLE

buildah add containerID '/myapp/app.conf' '/myapp/app.conf'

buildah add --chown myuser:mygroup containerID '/myapp/app.conf' '/myapp/app.conf'

buildah add --chmod 660 containerID '/myapp/app.conf' '/myapp/app.conf'

buildah add containerID '/home/myuser/myproject.go'

buildah add containerID '/home/myuser/myfiles.tar' '/tmp'

buildah add containerID '/tmp/workingdir' '/tmp/workingdir'

buildah add containerID 'https://github.com/containers/buildah/blob/main/README.md' '/tmp'

buildah add containerID 'passwd' 'certs.d' /etc

## FILES

### .containerignore or .dockerignore

If a .containerignore or .dockerignore file exists in the context directory,
`buildah add` reads its contents. If both exist, then .containerignore is used.

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
exclusions. The following is an example .containerignore file that uses this
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
