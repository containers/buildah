## buildah-copy "March 2017"

## NAME
buildah copy - Copies the contents of a file, URL, or directory into a container's working directory. 


## SYNOPSIS
**buildah** **copy** containerID **SRC** [**DEST**] 

## DESCRIPTION
Copies the contents of a file, URL, or a directory to a container's working directory.  If a local file appears to be an archive, its contents are extracted and copied instead of the archive file itself.  If a destination is not specified, the location of the source will be used for the destination.  



## EXAMPLE
**buildah copy containerID '/myapp/app.conf' '/myapp/app.conf' **
**buildah copy containerID '/home/myuser/myproject.go' **
**buildah copy containerID '/home/myuser/myfiles.tar' '/tmp' **
**buildah copy containerID '/tmp/workingdir' '/tmp/workingdir' **
**buildah copy containerID 'https://github.com/projectatomic/buildah' '/tmp' **
**buildah copy containerID 'passwd' 'certs.d' /etc **

## SEE ALSO
buildah(1)

