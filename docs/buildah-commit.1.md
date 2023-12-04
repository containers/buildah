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

The *image* value supports all transports from `containers-transports(5)`. If no transport is specified, the `containers-storage` (i.e., local storage) transport is used.

## RETURN VALUE
The image ID of the image that was created.  On error, 1 is returned and errno is returned.

## OPTIONS

**--add-file** *source[:destination]*

Read the contents of the file `source` and add it to the committed image as a
file at `destination`.  If `destination` is not specified, the path of `source`
will be used.  The new file will be owned by UID 0, GID 0, have 0644
permissions, and be given a current timestamp unless the **--timestamp** option
is also specified.  This option can be specified multiple times.

**--authfile** *path*

Path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json. If XDG_RUNTIME_DIR is not set, the default is /run/containers/$UID/auth.json. This file is created using `buildah login`.

If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

Note: You can also override the default path of the authentication file by setting the REGISTRY\_AUTH\_FILE
environment variable. `export REGISTRY_AUTH_FILE=path`

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) to connect to the registry.
The default certificates directory is _/etc/containers/certs.d_.

**--change**, **-c** *"INSTRUCTION"*

Apply the change to the committed image that would have been made if it had
been built using a Containerfile which included the specified instruction.
This option can be specified multiple times.

**--config** *filename*

Read a JSON-encoded version of an image configuration object from the specified
file, and merge the values from it with the configuration of the image being
committed.

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--cw** *options*

Produce an image suitable for use as a confidential workload running in a
trusted execution environment (TEE) using krun (i.e., *crun* built with the
libkrun feature enabled and invoked as *krun*).  Instead of the conventional
contents, the root filesystem of the image will contain an encrypted disk image
and configuration information for krun.

The value for *options* is a comma-separated list of key=value pairs, supplying
configuration information which is needed for producing the additional data
which will be included in the container image.

Recognized _keys_ are:

*attestation_url*: The location of a key broker / attestation server.
If a value is specified, the new image's workload ID, along with the passphrase
used to encrypt the disk image, will be registered with the server, and the
server's location will be stored in the container image.
At run-time, krun is expected to contact the server to retrieve the passphrase
using the workload ID, which is also stored in the container image.
If no value is specified, a *passphrase* value *must* be specified.

*cpus*: The number of virtual CPUs which the image expects to be run with at
run-time.  If not specified, a default value will be supplied.

*firmware_library*: The location of the libkrunfw-sev shared library.  If not
specified, `buildah` checks for its presence in a number of hard-coded
locations.

*memory*: The amount of memory which the image expects to be run with at
run-time, as a number of megabytes.  If not specified, a default value will be
supplied.

*passphrase*: The passphrase to use to encrypt the disk image which will be
included in the container image.
If no value is specified, but an *attestation_url* value is specified, a
randomly-generated passphrase will be used.
The authors recommend setting an *attestation_url* but not a *passphrase*.

*slop*: Extra space to allocate for the disk image compared to the size of the
container image's contents, expressed either as a percentage (..%) or a size
value (bytes, or larger units if suffixes like KB or MB are present), or a sum
of two or more such specifications separated by "+".  If not specified,
`buildah` guesses that 25% more space than the contents will be enough, but
this option is provided in case its guess is wrong.  If the specified or
computed size is less than 10 megabytes, it will be increased to 10 megabytes.

*type*: The type of trusted execution environment (TEE) which the image should
be marked for use with.  Accepted values are "SEV" (AMD Secure Encrypted
Virtualization - Encrypted State) and "SNP" (AMD Secure Encrypted
Virtualization - Secure Nested Paging).  If not specified, defaults to "SNP".

*workload_id*: A workload identifier which will be recorded in the container
image, to be used at run-time for retrieving the passphrase which was used to
encrypt the disk image.  If not specified, a semi-random value will be derived
from the base image's image ID.

**--disable-compression**, **-D**

Don't compress filesystem layers when building the image unless it is required
by the location where the image is being written.  This is the default setting,
because image layers are compressed automatically when they are pushed to
registries, and images being written to local storage would only need to be
decompressed again to be stored.  Compression can be forced in all cases by
specifying **--disable-compression=false**.

**--encrypt-layer** *layer(s)*

Layer(s) to encrypt: 0-indexed layer indices with support for negative indexing (e.g. 0 is the first layer, -1 is the last layer). If not defined, will encrypt all layers if encryption-key flag is specified.

**--encryption-key** *key*

The [protocol:keyfile] specifies the encryption protocol, which can be JWE (RFC7516), PGP (RFC4880), and PKCS7 (RFC2315) and the key material required for image encryption. For instance, jwe:/path/to/key.pem or pgp:admin@example.com or pkcs7:/path/to/x509-file.

**--format**, **-f** *[oci | docker]*

Control the format for the image manifest and configuration data.  Recognized
formats include *oci* (OCI image-spec v1.0, the default) and *docker* (version
2, using schema format 2 for the manifest).

Note: You can also override the default format by setting the BUILDAH_FORMAT
environment variable.  `export BUILDAH_FORMAT=docker`

**--identity-label** *bool-value*

Adds default identity label `io.buildah.version` if set. (default true).

**--iidfile** *ImageIDfile*

Write the image ID to the file.

**--manifest** "listName"

Name of the manifest list to which the built image will be added. Creates the manifest list
if it does not exist. This option is useful for building multi architecture images.

**--omit-history** *bool-value*

Omit build history information in the built image. (default false).

This option is useful for the cases where end users explicitly
want to set `--omit-history` to omit the optional `History` from
built images or when working with images built using build tools that
do not include `History` information in their images.

**--pull**

When the *--pull* flag is enabled or set explicitly to `true` (with
*--pull=true*), attempt to pull the latest versions of SBOM scanner images from
the registries listed in registries.conf if a local SBOM scanner image does not
exist or the image in the registry is newer than the one in local storage.
Raise an error if the SBOM scanner image is not in any listed registry and is
not present locally.

If the flag is disabled (with *--pull=false*), do not pull SBOM scanner images
from registries, use only local versions. Raise an error if a SBOM scanner
image is not present locally.

If the pull flag is set to `always` (with *--pull=always*), pull SBOM scanner
images from the registries listed in registries.conf.  Raise an error if a SBOM
scanner image is not found in the registries, even if an image with the same
name is present locally.

If the pull flag is set to `missing` (with *--pull=missing*), pull SBOM scanner
images only if they could not be found in the local containers storage.  Raise
an error if no image could be found and the pull fails.

If the pull flag is set to `never` (with *--pull=never*), do not pull SBOM
scanner images from registries, use only the local versions.  Raise an error if
the image is not present locally.

**--quiet**, **-q**

When writing the output image, suppress progress output.

**--rm**
Remove the working container and its contents after creating the image.
Default leaves the container and its content in place.

**--sbom** *preset*

Generate SBOMs (Software Bills Of Materials) for the output image by scanning
the working container and build contexts using the named combination of scanner
image, scanner commands, and merge strategy.  Must be specified with one or
more of **--sbom-image-output**, **--sbom-image-purl-output**, **--sbom-output**,
and **--sbom-purl-output**.  Recognized presets, and the set of options which
they equate to:

 - "syft", "syft-cyclonedx":
     --sbom-scanner-image=ghcr.io/anchore/syft
     --sbom-scanner-command="/syft scan -q dir:{ROOTFS} --output cyclonedx-json={OUTPUT}"
     --sbom-scanner-command="/syft scan -q dir:{CONTEXT} --output cyclonedx-json={OUTPUT}"
     --sbom-merge-strategy=merge-cyclonedx-by-component-name-and-version
 - "syft-spdx":
     --sbom-scanner-image=ghcr.io/anchore/syft
     --sbom-scanner-command="/syft scan -q dir:{ROOTFS} --output spdx-json={OUTPUT}"
     --sbom-scanner-command="/syft scan -q dir:{CONTEXT} --output spdx-json={OUTPUT}"
     --sbom-merge-strategy=merge-spdx-by-package-name-and-versioninfo
 - "trivy", "trivy-cyclonedx":
     --sbom-scanner-image=ghcr.io/aquasecurity/trivy
     --sbom-scanner-command="trivy filesystem -q {ROOTFS} --format cyclonedx --output {OUTPUT}"
     --sbom-scanner-command="trivy filesystem -q {CONTEXT} --format cyclonedx --output {OUTPUT}"
     --sbom-merge-strategy=merge-cyclonedx-by-component-name-and-version
 - "trivy-spdx":
     --sbom-scanner-image=ghcr.io/aquasecurity/trivy
     --sbom-scanner-command="trivy filesystem -q {ROOTFS} --format spdx-json --output {OUTPUT}"
     --sbom-scanner-command="trivy filesystem -q {CONTEXT} --format spdx-json --output {OUTPUT}"
     --sbom-merge-strategy=merge-spdx-by-package-name-and-versioninfo

**--sbom-image-output** *path*

When generating SBOMs, store the generated SBOM in the specified path in the
output image.  There is no default.

**--sbom-image-purl-output** *path*

When generating SBOMs, scan them for PURL ([package
URL](https://github.com/package-url/purl-spec/blob/master/PURL-SPECIFICATION.rst))
information, and save a list of found PURLs to the named file in the local
filesystem.  There is no default.

**--sbom-merge-strategy** *method*

If more than one **--sbom-scanner-command** value is being used, use the
specified method to merge the output from later commands with output from
earlier commands.  Recognized values include:

 - cat
     Concatenate the files.
 - merge-cyclonedx-by-component-name-and-version
     Merge the "component" fields of JSON documents, ignoring values from
     documents when the combination of their "name" and "version" values is
     already present.  Documents are processed in the order in which they are
     generated, which is the order in which the commands that generate them
     were specified.
 - merge-spdx-by-package-name-and-versioninfo
     Merge the "package" fields of JSON documents, ignoring values from
     documents when the combination of their "name" and "versionInfo" values is
     already present.  Documents are processed in the order in which they are
     generated, which is the order in which the commands that generate them
     were specified.

**--sbom-output** *file*

When generating SBOMs, store the generated SBOM in the named file on the local
filesystem.  There is no default.

**--sbom-purl-output** *file*

When generating SBOMs, scan them for PURL ([package
URL](https://github.com/package-url/purl-spec/blob/master/PURL-SPECIFICATION.rst))
information, and save a list of found PURLs to the named file in the local
filesystem.  There is no default.

**--sbom-scanner-command** *image*

Generate SBOMs by running the specified command from the scanner image.  If
multiple commands are specified, they are run in the order in which they are
specified.  These text substitutions are performed:
  - {ROOTFS}
      The root of the built image's filesystem, bind mounted.
  - {CONTEXT}
      The build context and additional build contexts, bind mounted.
  - {OUTPUT}
      The name of a temporary output file, to be read and merged with others or copied elsewhere.

**--sbom-scanner-image** *image*

Generate SBOMs using the specified scanner image.

**--sign-by** *fingerprint*

Sign the new image using the GPG key that matches the specified fingerprint.

**--squash**

Squash all of the new image's layers (including those inherited from a base image) into a single new layer.

**--timestamp** *seconds*

Set the create timestamp to seconds since epoch to allow for deterministic builds (defaults to current time).
By default, the created timestamp is changed and written into the image manifest with every commit,
causing the image's sha256 hash to be different even if the sources are exactly the same otherwise.
When --timestamp is set, the created timestamp is always set to the time specified and therefore not changed, allowing the image's sha256 to remain the same. All files committed to the layers of the image will be created with the timestamp.

**--tls-verify** *bool-value*

Require HTTPS and verification of certificates when talking to container registries (defaults to true).  TLS verification cannot be used when talking to an insecure registry.

**--unsetenv** *env*

Unset environment variables from the final image.

## EXAMPLE

This example saves an image based on the container.
 `buildah commit containerID newImageName`

This example saves an image named newImageName based on the container and removes the working container.
 `buildah commit --rm containerID newImageName`

This example commits to an OCI archive file named /tmp/newImageName based on the container.
 `buildah commit containerID oci-archive:/tmp/newImageName`

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

### Building an multi-architecture image using the --manifest option (requires emulation software)

```
#!/bin/sh
build() {
	ctr=$(./bin/buildah from --arch $1 ubi8)
	./bin/buildah run $ctr dnf install -y iputils
	./bin/buildah commit --manifest ubi8ping $ctr
}
build arm
build amd64
build s390x
```

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
buildah(1), buildah-images(1), containers-policy.json(5), containers-registries.conf(5), containers-transports(5)
