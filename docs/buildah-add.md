## buildah-add "1" "March 2017" "buildah"

## NAME
buildah add - Add the contents of a file, URL, or a directory to a container.

## SYNOPSIS
**buildah** **add** **containerID** **SRC** [[...] **DEST**]

## DESCRIPTION
Adds the contents of a file, URL, or a directory to a container's working
directory or a specified location in the container.  If a local source file
appears to be an archive, its contents are extracted and added instead of the
archive file itself.  If a local directory is specified as a source, its
*contents* are copied to the destination.

## EXAMPLE

buildah add containerID '/myapp/app.conf' '/myapp/app.conf'

buildah add containerID '/home/myuser/myproject.go'

buildah add containerID '/home/myuser/myfiles.tar' '/tmp'

buildah add containerID '/tmp/workingdir' '/tmp/workingdir'

buildah add containerID 'https://github.com/projectatomic/buildah/blob/master/README.md' '/tmp'

buildah add containerID 'passwd' 'certs.d' /etc

## SEE ALSO
buildah(1)
