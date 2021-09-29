# buildah-rmi "1" "March 2017" "buildah"

## NAME

buildah\-rmi - Removes one or more images.

## SYNOPSIS

**buildah rmi** [*image* ...]

## DESCRIPTION

Removes one or more locally stored images.
Passing an argument _image_ deletes it, along with any of its dangling (untagged) parent images.

## LIMITATIONS

* If the image was pushed to a directory path using the 'dir:' transport,
  the rmi command can not remove the image. Instead, standard file system
  commands should be used.

* If _imageID_ is a name, but does not include a registry name, buildah will
  attempt to find and remove the named image using the registry name _localhost_,
  if no such image is found, it will search for the intended image by attempting
  to expand the given name using the names of registries provided in the system's
  registries configuration file, registries.conf.

* If the _imageID_ refers to a *manifest list* or *image index*, this command
  will ***not*** do what you expect!  This command will remove the images
  associated with the *manifest list* or *index* (not the manifest list/image index
  itself). To remove that, use the `buildah manifest rm` subcommand instead.

## OPTIONS

**--all**, **-a**

All local images will be removed from the system that do not have containers using the image as a reference image.
An image name or id cannot be provided when this option is used. Read/Only images configured by modifying the "additionalimagestores" in the /etc/containers/storage.conf file, can not be removed.

**--prune**, **-p**

All local images will be removed from the system that do not have a tag and do not have a child image pointing to them.
An image name or id cannot be provided when this option is used.

**--force**, **-f**

This option will cause Buildah to remove all containers that are using the image before removing the image from the system.

## EXAMPLE

buildah rmi imageID

buildah rmi --all

buildah rmi --all --force

buildah rmi --prune

buildah rmi --force imageID

buildah rmi imageID1 imageID2 imageID3

## Files

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

**storage.conf** (`/etc/containers/storage.conf`)

storage.conf is the storage configuration file for all tools using containers/storage

The storage configuration file specifies all of the available container storage options for tools using shared container storage.

## SEE ALSO

buildah(1), containers-registries.conf(5), containers-storage.conf(5)
