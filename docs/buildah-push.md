# buildah-push "1" "June 2017" "buildah"

## NAME
buildah\-push - Push an image from local storage to elsewhere.

## SYNOPSIS
**buildah push** [*options*] *image* [*destination*]

## DESCRIPTION
Pushes an image from local storage to a specified destination, decompressing
and recompessing layers as needed.

## imageID
Image stored in local container/storage

## DESTINATION

 The DESTINATION is a location to store container images. If omitted, the source image parameter will be reused as destination.

 The Image "DESTINATION" uses a "transport":"details" format. Multiple transports are supported:

  **dir:**_path_
  An existing local directory _path_ storing the manifest, layer tarballs and signatures as individual files. This is a non-standardized format, primarily useful for debugging or noninvasive container inspection.

  **docker://**_docker-reference_
  An image in a registry implementing the "Docker Registry HTTP API V2". By default, uses the authorization state in `$XDG\_RUNTIME\_DIR/containers/auth.json`, which is set using `(buildah login)`. If the authorization state is not found there, `$HOME/.docker/config.json` is checked, which is set using `(docker login)`.
  If _docker-reference_ does not include a registry name, the image will be pushed to a registry running on *localhost*.

  **docker-archive:**_path_[**:**_docker-reference_]
  An image is stored in the `docker save` formatted file.  _docker-reference_ is only used when creating such a file, and it must not contain a digest.

  **docker-daemon:**_docker-reference_
  An image _docker_reference_ stored in the docker daemon internal storage. If _docker_reference_ does not begin with a valid registry name (a domain name containing "." or the reserved name "localhost") then the default registry name "docker.io" will be prepended. _docker_reference_ must contain either a tag or a digest. Alternatively, when reading images, the format can also be docker-daemon:algo:digest (an image ID).

  **oci:**_path_**:**_tag_
  An image _tag_ in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **oci-archive:**_path_**:**_tag_
  An image _tag_ in a tar archive compliant with "Open Container Image Layout Specification" at _path_.

If the transport part of DESTINATION is omitted, "docker://" is assumed.

## OPTIONS

**--authfile** *path*

Path of the authentication file. Default is ${XDG\_RUNTIME\_DIR}/containers/auth.json, which is set using `buildah login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) to connect to the registry.
Default certificates directory is _/etc/containers/certs.d_.

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--digestfile** *Digestfile*

After copying the image, write the digest of the resulting image to the file.

**--disable-compression, -D**

Don't compress copies of filesystem layers which will be pushed.

**--encryption-key** *key*

The [protocol:keyfile] specifies the encryption protocol, which can be JWE (RFC7516), PGP (RFC4880), and PKCS7 (RFC2315) and the key material required for image encryption. For instance, jwe:/path/to/key.pem or pgp:admin@example.com or pkcs7:/path/to/x509-file.

**--format, -f**

Manifest Type (oci, v2s1, or v2s2) to use when saving image to directory using the 'dir:' transport (default is manifest type of source)

**--quiet, -q**

When writing the output image, suppress progress output.

**--remove-signatures**

Don't copy signatures when pushing images.

**--sign-by** *fingerprint*

Sign the pushed image using the GPG key that matches the specified fingerprint.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)

## EXAMPLE

This example pushes the image specified by the imageID to a local directory in docker format.

 `# buildah push imageID dir:/path/to/image`

This example pushes the image specified by the imageID to a local directory in oci format.

 `# buildah push imageID oci:/path/to/layout:image:tag`

This example pushes the image specified by the imageID to a tar archive in oci format.

  `# buildah push imageID oci-archive:/path/to/archive:image:tag`

This example pushes the image specified by the imageID to a container registry named registry.example.com.

 `# buildah push imageID docker://registry.example.com/repository:tag`

This example pushes the image specified by the imageID to a container registry named registry.example.com and saves the digest in the specified digestfile.

 `# buildah push --digestfile=/tmp/mydigest imageID docker://registry.example.com/repository:tag`

This example works like **docker push**, assuming *registry.example.com/my_image* is a local image.

 `# buildah push registry.example.com/my_image`

This example pushes the image specified by the imageID to a private container registry named registry.example.com with authentication from /tmp/auths/myauths.json.

 `# buildah push --authfile /tmp/auths/myauths.json imageID docker://registry.example.com/repository:tag`

This example pushes the image specified by the imageID and puts into the local docker container store.

 `# buildah push imageID docker-daemon:image:tag`

This example pushes the image specified by the imageID and puts it into the registry on the localhost while turning off tls verification.
 `# buildah push --tls-verify=false imageID docker://localhost:5000/my-imageID`

This example pushes the image specified by the imageID and puts it into the registry on the localhost using credentials and certificates for authentication.
 `# buildah push --cert-dir ~/auth --tls-verify=true --creds=username:password imageID docker://localhost:5000/my-imageID`

## ENVIRONMENT

**BUILD\_REGISTRY\_SOURCES**

BUILD\_REGISTRY\_SOURCES, if set, is treated as a JSON object which contains
lists of registry names under the keys `insecureRegistries`,
`blockedRegistries`, and `allowedRegistries`.

When pushing an image to a registry, if the portion of the destination image
name that corresponds to a registry is compared to the items in the
`blockedRegistries` list, and if it matches any of them, the push attempt is
denied.  If there are registries in the `allowedRegistries` list, and the
portion of the name that corresponds to the registry is not in the list, the
push attempt is denied.

**TMPDIR**
The TMPDIR environment variable allows the user to specify where temporary files
are stored while pulling and pushing images.  Defaults to '/var/tmp'.

## FILES

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

**policy.json** (`/etc/containers/policy.json`)

Signature policy file.  This defines the trust policy for container images.  Controls which container registries can be used for image, and whether or not the tool should trust the images.

## SEE ALSO
buildah(1), buildah-login(1), containers-policy.json(5), docker-login(1), containers-registries.conf(5)
