## buildah "1" "March 2017" "buildah"

## NAME
buildah - A command line tool to facilitate working with containers and using them to build images.

## DESCRIPTION
The buildah package provides a command line tool which can be used to:

    * Create a working container, either from scratch or using an image as a starting point.
    * Mount a working container's root filesystem for manipulation.
    * Unmount a working container's root filesystem.
    * Use the updated contents of a container's root filesystem as a filesystem layer to create a new image.
    * Delete a working container or an image.

## SEE ALSO
| Command               | Description |
| --------------------- | --------------------------------------------------- |
| buildah-add(1)        | Add the contents of a file, URL, or a directory to the container. |
| buildah-commit(1)     | Create an image from a working container. |
| buildah-config(1)     | Update image configuration settings. |
| buildah-containers(1) | List the working containers and their base images. |
| buildah-copy(1)       | Copies the contents of a file, URL, or directory into a container's working directory. |
| buildah-from(1)       | Creates a new working container, either from scratch or using a specified image as a starting point. |
| buildah-images(1)     | List images in local storage. |
| buildah-mount(1)      | Mount the working container's root filesystem. |
| buildah-rm(1)         | Removes one or more working containers. |
| buildah-rmi(1)        | Removes one or more images. |
| buildah-run(1)        | Run a command inside of the container. |
| buildah-umount(1)     | Unmount a working container's root file system. |
