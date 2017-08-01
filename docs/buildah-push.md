## buildah-push"1" "June 2017" "buildah"

## NAME
buildah push - Push an image from local storage to elsewhere.

## SYNOPSIS
**buildah** **push** [*options* [...]] **imageID** [**destination**]

## DESCRIPTION
Pushes an image from local storage to a specified destination, decompressing
and recompessing layers as needed.

## imageID
Image stored in local container/storage

## DESTINATION

 The DESTINATION is a location to store container images
 The Image "DESTINATION" uses a "transport":"details" format.

 Multiple transports are supported:

  **dir:**_path_
  An existing local directory _path_ storing the manifest, layer tarballs and signatures as individual files. This is a non-standardized format, primarily useful for debugging or noninvasive container inspection.

  **docker://**_docker-reference_
  An image in a registry implementing the "Docker Registry HTTP API V2". By default, uses the authorization state in `$HOME/.docker/config.json`, which is set e.g. using `(docker login)`.

  **docker-archive:**_path_[**:**_docker-reference_]
  An image is stored in the `docker save` formatted file.  _docker-reference_ is only used when creating such a file, and it must not contain a digest.

  **docker-daemon:**_docker-reference_
  An image _docker-reference_ stored in the docker daemon internal storage.  _docker-reference_ must contain either a tag or a digest.  Alternatively, when reading images, the format can also be docker-daemon:algo:digest (an image ID).

  **oci:**_path_**:**_tag_
  An image _tag_ in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **ostree:**_image_[**@**_/absolute/repo/path_]
  An image in local OSTree repository.  _/absolute/repo/path_ defaults to _/ostree/repo_.

## OPTIONS

**--disable-compression, -D**

Don't compress copies of filesystem layers which will be pushed.

**--signature-policy**

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**--quiet**

When writing the output image, suppress progress output.

## EXAMPLE

This example extracts the imageID image to a local directory in docker format.

 `# buildah push imageID dir:/path/to/image`

This example extracts the imageID image to a local directory in oci format.

 `# buildah push imageID oci:/path/to/layout`

This example extracts the imageID image to a container registry named registry.example.com

 `# buildah push imageID docker://registry.example.com/repository:tag`

This example extracts the imageID image and puts into the local docker container store

 `# buildah push imageID docker-daemon:image:tag`

## SEE ALSO
buildah(1)
