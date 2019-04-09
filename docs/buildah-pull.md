# buildah-pull "1" "July 2018" "buildah"

## NAME
buildah\-pull - Pull an image from a registry.

## SYNOPSIS
**buildah pull** [*options*] *image*

## DESCRIPTION
Pulls an image based upon the specified image name.  Image names
use a "transport":"details" format.

Multiple transports are supported:

  **dir:**_path_
  An existing local directory _path_ containing the manifest, layer tarballs, and signatures in individual files. This is a non-standardized format, primarily useful for debugging or noninvasive image inspection.

  **docker://**_docker-reference_ (Default)
  An image in a registry implementing the "Docker Registry HTTP API V2". By default, uses the authorization state in `$XDG\_RUNTIME\_DIR/containers/auth.json`, which is set using `(buildah login)`. If the authorization state is not found there, `$HOME/.docker/config.json` is checked, which is set using `(docker login)`.
  If _docker-reference_ does not include a registry name, *localhost* will be consulted first, followed by any registries named in the registries configuration.

  **docker-archive:**_path_
  An image is retrieved as a `docker load` formatted file.

  **docker-daemon:**_docker-reference_
  An image _docker-reference_ stored in the docker daemon's internal storage.  _docker-reference_ must include either a tag or a digest.  Alternatively, when reading images, the format can also be docker-daemon:algo:digest (an image ID).

  **oci:**_path_**:**_tag_**
  An image tag in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **oci-archive:**_path_**:**_tag_
  An image _tag_ in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **ostree:**_image_[**@**_/absolute/repo/path_]
  An image in local OSTree repository.  _/absolute/repo/path_ defaults to _/ostree/repo_.

### DEPENDENCIES

Buildah resolves the path to the registry to pull from by using the /etc/containers/registries.conf
file, registries.conf(5).  If the `buildah pull` command fails with an "image not known" error,
first verify that the registries.conf file is installed and configured appropriately.

## RETURN VALUE
The image ID of the image that was pulled.  On error 1 is returned.

## OPTIONS

**--all-tags, a**

All tagged images in the repository will be pulled.

**--authfile** *path*

Path of the authentication file. Default is ${XDG\_RUNTIME\_DIR}/containers/auth.json, which is set using `buildah login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) to connect to the registry.
The default certificates directory is _/etc/containers/certs.d_.

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--quiet, -q**

If an image needs to be pulled from the registry, suppress progress output.

**--shm-size**=""

Size of `/dev/shm`. The format is `<number><unit>`. `number` must be greater than `0`.
Unit is optional and can be `b` (bytes), `k` (kilobytes), `m`(megabytes), or `g` (gigabytes).
If you omit the unit, the system uses bytes. If you omit the size entirely, the system uses `64m`.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)


## EXAMPLE

buildah pull imagename

buildah pull docker://myregistry.example.com/imagename

buildah pull docker-daemon:imagename:imagetag

buildah pull docker-archive:filename

buildah pull oci-archive:filename

buildah pull dir:directoryname

buildah pull --tls-verify=false myregistry/myrepository/imagename:imagetag

buildah pull --creds=myusername:mypassword --cert-dir ~/auth myregistry/myrepository/imagename:imagetag

buildah pull --authfile=/tmp/auths/myauths.json myregistry/myrepository/imagename:imagetag


## Files

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

**policy.json** (`/etc/containers/policy.json`)

Signature policy file.  This defines the trust policy for container images.  Controls which container registries can be used for image, and whether or not the tool should trust the images.

## SEE ALSO
buildah(1), buildah-from(1), buildah-login(1), docker-login(1), policy.json(5), registries.conf(5)
