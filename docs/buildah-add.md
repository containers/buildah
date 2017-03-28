## buildah-add "March 2017"

## NAME
buildah add - Add the contents of a file, URL, or a directory to the container. 


## SYNOPSIS
**buildah** **add** **containerID** **SRC** [**DEST**] 

## DESCRIPTION
Adds the contents of a file, URL, or a directory to a container's working directory.  If a local file appears to be an archive, its contents are extracted and added instead of the archive file itself.  If a destination is not specified, the location of the source will be used for the destination.  

## EXAMPLE
**buildah add containerID '/myapp/app.conf' '/myapp/app.conf' **
**buildah add containerID '/home/myuser/myproject.go' **
**buildah add containerID '/home/myuser/myfiles.tar' '/tmp' **
**buildah add containerID '/tmp/workingdir' '/tmp/workingdir' **
**buildah add containerID 'https://github.com/projectatomic/buildah' '/tmp' **
**buildah add containerID 'passwd' 'certs.d' /etc **

## SEE ALSO
buildah(1)

