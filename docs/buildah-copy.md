# buildah-copy "1" "March 2017" "buildah"

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

**--chown** *owner*:*group*

Sets the user and group ownership of the destination content.

**--quiet**

Refrain from printing a digest of the copied content.

## EXAMPLE

buildah copy containerID '/myapp/app.conf' '/myapp/app.conf'

buildah copy --chown myuser:mygroup containerID '/myapp/app.conf' '/myapp/app.conf'

buildah copy containerID '/home/myuser/myproject.go'

buildah copy containerID '/home/myuser/myfiles.tar' '/tmp'

buildah copy containerID '/tmp/workingdir' '/tmp/workingdir'

buildah copy containerID 'https://github.com/containers/buildah' '/tmp'

buildah copy containerID 'passwd' 'certs.d' /etc

## FILES

### `.dockerignore`

If the file .dockerignore exists in the context directory, `buildah copy` reads
its contents. Buildah uses the content to exclude files and directories from
the context directory, when copying content into the image.

Users can specify a series of Unix shell globals in a .dockerignore file to
identify files to exclude.

Buildah supports a special wildcard string `**` which matches any number of
directories (including zero). For example, **/*.go will exclude all files that
end with .go that are found in all directories.

Example .dockerignore file:

```
# comment
*/temp*
**/temp*
temp?
```

`*/temp*`
Excludes files and directories whose names start with temp in any immediate subdirectory of the root. For example, the plain file /somedir/temporary.txt is excluded, as is the directory /somedir/temp.

`**/temp*`
Excludes files and directories starting with temp from any subdirectory.

`temp?`
Excludes files and directories in the root directory whose names are a one-character extension of temp. For example, /tempa and /tempb are excluded.

Lines starting with ! (exclamation mark) can be used to make exceptions to
exclusions. The following is an example .dockerignore file that uses this
mechanism:
```
*.md
!README.md
```

All markdown files except README.md are excluded from the context.

For more information see:

https://docs.docker.com/engine/reference/builder/#dockerignore-file

## SEE ALSO
buildah(1)
