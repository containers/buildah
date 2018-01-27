## buildah-from "1" "March 2017" "buildah"

## NAME
buildah from - Creates a new working container, either from scratch or using a specified image as a starting point.

## SYNOPSIS
**buildah** **from** [*options* [...]] *imageName*

## DESCRIPTION
Creates a working container based upon the specified image name.  If the
supplied image name is "scratch" a new empty container is created. Image names
uses a "transport":"details" format.

Multiple transports are supported:

  **dir:**_path_
  An existing local directory _path_ retrieving the manifest, layer tarballs and signatures as individual files. This is a non-standardized format, primarily useful for debugging or noninvasive container inspection.

  **docker://**_docker-reference_ (Default)
  An image in a registry implementing the "Docker Registry HTTP API V2". By default, uses the authorization state in `$XDG_RUNTIME_DIR/containers/auth.json`, which is set using `(kpod login)`. If the authorization state is not found there, `$HOME/.docker/config.json` is checked, which is set using `(docker login)`.

  **docker-archive:**_path_
  An image is retrieved as a `docker load` formatted file.

  **docker-daemon:**_docker-reference_
  An image _docker-reference_ stored in the docker daemon internal storage.  _docker-reference_ must contain either a tag or a digest.  Alternatively, when reading images, the format can also be docker-daemon:algo:digest (an image ID).

  **oci:**_path_**:**_tag_
  An image _tag_ in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **ostree:**_image_[**@**_/absolute/repo/path_]
  An image in local OSTree repository.  _/absolute/repo/path_ defaults to _/ostree/repo_.

## RETURN VALUE
The container ID of the container that was created.  On error, -1 is returned and errno is returned.

## OPTIONS

**--authfile** *path*

Path of the authentication file. Default is ${XDG_RUNTIME\_DIR}/containers/auth.json, which is set using `kpod login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

**--cert-dir** *path*

Use certificates at *path* (*.crt, *.cert, *.key) to connect to the registry

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--name** *name*

A *name* for the working container

**--pull**

Pull the image if it is not present.  If this flag is disabled (with
*--pull=false*) and the image is not present, the image will not be pulled.
Defaults to *true*.

**--pull-always**

Pull the image even if a version of the image is already present.

**--quiet**

If an image needs to be pulled from the registry, suppress progress output.

**--signature-policy** *signaturepolicy*

Pathname of a signature policy file to use.  It is not recommended that this
option be used, as the default behavior of using the system-wide default policy
(frequently */etc/containers/policy.json*) is most often preferred.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)

## EXAMPLE

buildah from imagename --pull

buildah from docker://myregistry.example.com/imagename --pull

buildah from imagename --signature-policy /etc/containers/policy.json

buildah from docker://myregistry.example.com/imagename --pull-always --name "mycontainer"

buildah from myregistry/myrepository/imagename:imagetag --tls-verify=false

buildah from myregistry/myrepository/imagename:imagetag --creds=myusername:mypassword --cert-dir ~/auth

buildah from myregistry/myrepository/imagename:imagetag --authfile=/tmp/auths/myauths.json

## SEE ALSO
buildah(1), kpod-login(1), docker-login(1)
