# buildah-manifest-push "1" "September 2019" "buildah"

## NAME

buildah\-manifest\-push - Push a manifest list or image index to a registry.

## SYNOPSIS

**buildah manifest push** [options...] *listNameOrIndexName* *transport:details*

## DESCRIPTION

Pushes a manifest list or image index to a registry.

## RETURN VALUE

The list image's ID and the digest of the image's manifest.

## OPTIONS

**--all**

Push the images mentioned in the manifest list or image index, in addition to
the list or index itself.

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

**--digestfile** *Digestfile*

After copying the image, write the digest of the resulting image to the file.

**--format**, **-f**

Manifest list type (oci or v2s2) to use when pushing the list (default is oci).

**--quiet**, **-q**

Don't output progress information when pushing lists.

**--remove-signatures**

Don't copy signatures when pushing images.

**--rm**

Delete the manifest list or image index from local storage if pushing succeeds.

**--sign-by** *fingerprint*

Sign the pushed images using the GPG key that matches the specified fingerprint.

**--tls-verify** *bool-value*

Require HTTPS and verification of certificates when talking to container registries (defaults to true).  TLS verification cannot be used when talking to an insecure registry.

## EXAMPLE

```
buildah manifest push mylist:v1.11 registry.example.org/mylist:v1.11
```

## SEE ALSO
buildah(1), buildah-login(1), buildah-manifest(1), buildah-manifest-create(1), buildah-manifest-add(1), buildah-manifest-remove(1), buildah-manifest-annotate(1), buildah-manifest-inspect(1), buildah-rmi(1), docker-login(1)
