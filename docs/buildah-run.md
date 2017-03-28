## buildah-run "March 2017"

## NAME
buildah run - Run a command inside of the container. 


## SYNOPSIS
**buildah** **run** **containerID** [*command options* [...]] 

## DESCRIPTION
Launches a container and runs the specified command in that container using the container's root filesystem as a root filesystem, using configuration settings inherited from the container's image or as specified using previous calls to the buildah-config command.

## OPTIONS

**--runtime**
The *path* to an alternate runtime.

**--runtime-flag**
Adds global flags for the containter rutime.


## EXAMPLE
**buildah run containerID 'ps -auxw' **

## SEE ALSO
buildah(1)

