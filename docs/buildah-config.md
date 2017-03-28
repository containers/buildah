## buildah-config "March 2017"

## NAME
buildah config - Update image configuration settings. 


## SYNOPSIS
**buildah** **config** **containerID** [*command options* [...]] 

## DESCRIPTION
Update a number of the image's configuration settings such as the
name, author, label and more. 

## OPTIONS

**--annotation *annotation* **
Image *annotation* e.g. annotation=*annotation*, for the target image

**--arch *architecture* **
*architecture* of the target image

**--author *author* **
Image *author* contact information

**--cmd *command* **
*command* for containers based on image

**--created-by *created* **dd
Description of how the image was *created* (default: "manual edits")

**--entrypoint *entry* **
*entry* point for containers based on image

**--env *envar* **
Environment variable (*envar*) to set when running containers based on image

**--label *label* **
Image configuration *label* e.g. label=*label*

**--os *operating system* **
Image target *operating system*

**--port *port* **
*port* to expose when running containers based on image

**--user *user* **
*user* to run containers based on image as

**--volume *volume* **
*volume* to create for containers based on image

**--workingdir *directory* **
Initial working *directory* for containers based on image

## EXAMPLE
**buildah config containerID --author='Jane Austen' --workingdir='/etc/mycontainers' **

## SEE ALSO
buildah(1)

