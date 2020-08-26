# buildah-commit "1" "March 2017" "buildah"

## NAME
buildah\-commit - Create an image from a working container.

## SYNOPSIS
**buildah commit** [*options*] *container* [*image*]

## DESCRIPTION
Writes a new image using the specified container's read-write layer and if it
is based on an image, the layers of that image.  If *image* does not begin
with a registry name component, `localhost` will be added to the name.  If
*image* is not provided, the image will have no name.  When an image has no
name, the `buildah images` command will display `<none>` in the `REPOSITORY` and
`TAG` columns.

## RETURN VALUE
The image ID of the image that was created.  On error, 1 is returned and errno is returned.

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

**--disable-compression, -D**

Don't compress filesystem layers when building the image unless it is required
by the location where the image is being written.  This is the default setting,
because image layers are compressed automatically when they are pushed to
registries, and images being written to local storage would only need to be
decompressed again to be stored.  Compression can be forced in all cases by
specifying **--disable-compression=false**.

**--format**

Control the format for the image manifest and configuration data.  Recognized
formats include *oci* (OCI image-spec v1.0, the default) and *docker* (version
2, using schema format 2 for the manifest).

Note: You can also override the default format by setting the BUILDAH\_FORMAT
environment variable.  `export BUILDAH\_FORMAT=docker`

**--iidfile** *ImageIDfile*

Write the image ID to the file.

**--quiet**

When writing the output image, suppress progress output.

**--rm**
Remove the working container and its contents after creating the image.
Default leaves the container and its content in place.

**--sign-by** *fingerprint*

Sign the new image using the GPG key that matches the specified fingerprint.

**--squash**

Squash all of the new image's layers (including those inherited from a base image) into a single new layer.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)

**--timestamp** *secconds*

Set the create timestamp to seconds since epoch to allow for deterministic builds (defaults to current time).
By default, the created timestamp is changed and written into the image manifest with every commit,
causing the image's sha256 hash to be different even if the sources are exactly the same otherwise.
When --timestamp is set, the created timestamp is always set to the time specified and therefore not changed, allowing the image's sha256 to remain the same. All files committed to the layers of the image will be created with the timestamp.

## EXAMPLE

This example saves an image based on the container.
 `buildah commit containerID newImageName`

This example saves an image named newImageName based on the container.
 `buildah commit --rm containerID newImageName`

This example saves an image with no name, removes the working container, and creates a new container using the image's ID.
 `buildah from $(buildah commit --rm containerID)`

This example saves an image based on the container disabling compression.
 `buildah commit --disable-compression containerID`

This example saves an image named newImageName based on the container disabling compression.
 `buildah commit --disable-compression containerID newImageName`

This example commits the container to the image on the local registry while turning off tls verification.
 `buildah commit --tls-verify=false containerID docker://localhost:5000/imageId`

This example commits the container to the image on the local registry using credentials and certificates for authentication.
 `buildah commit --cert-dir ~/auth  --tls-verify=true --creds=username:password containerID docker://localhost:5000/imageId`

This example commits the container to the image on the local registry using credentials from the /tmp/auths/myauths.json file and certificates for authentication.
 `buildah commit --authfile /tmp/auths/myauths.json --cert-dir ~/auth  --tls-verify=true --creds=username:password containerID docker://localhost:5000/imageName`

This example saves an image based on the container, but stores dates based on epoch time.
`buildah commit --timestamp=0 containerID newImageName`

## ENVIRONMENT

**BUILD\_REGISTRY\_SOURCES**

BUILD\_REGISTRY\_SOURCES, if set, is treated as a JSON object which contains
lists of registry names under the keys `insecureRegistries`,
`blockedRegistries`, and `allowedRegistries`.

When committing an image, if the image is to be given a name, the portion of
the name that corresponds to a registry is compared to the items in the
`blockedRegistries` list, and if it matches any of them, the commit attempt is
denied.  If there are registries in the `allowedRegistries` list, and the
portion of the name that corresponds to the registry is not in the list, the
commit attempt is denied.

**TMPDIR**
The TMPDIR environment variable allows the user to specify where temporary files
are stored while pulling and pushing images.  Defaults to '/var/tmp'.

## FILES

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

**policy.json** (`/etc/containers/policy.json`)

Signature policy file.  This defines the trust policy for container images.  Controls which container registries can be used for image, and whether or not the tool should trust the images.

## SEE ALSO
buildah(1), buildah-images(1), containers-policy.json(5), containers-registries.conf(5)
