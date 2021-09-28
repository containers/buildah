# buildah-manifest-add "1" "September 2019" "buildah"

## NAME

buildah\-manifest\-add - Add an image to a manifest list or image index.

## SYNOPSIS

**buildah manifest add** *listNameOrIndexName* *imageName*

## DESCRIPTION

Adds the specified image to the specified manifest list or image index.

## RETURN VALUE

The list image's ID and the digest of the image's manifest.

## OPTIONS

**--all**

If the image which should be added to the list or index is itself a list or
index, add all of the contents to the local list.  By default, only one image
from such a list or index will be added to the list or index.  Combining
*--all* with any of the other options described below is NOT recommended.

**--annotation** *annotation=value*

Set an annotation on the entry for the newly-added image.

**--arch**

Override the architecture which the list or index records as a requirement for
the image.  If *imageName* refers to a manifest list or image index, the
architecture information will be retrieved from it.  Otherwise, it will be
retrieved from the image's configuration information.

**--authfile** *path*

Path of the authentication file. Default is ${XDG_\RUNTIME\_DIR}/containers/auth.json. If XDG_RUNTIME_DIR is not set, the default is /run/containers/$UID/auth.json. This file is created using using `buildah login`.

If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

Note: You can also override the default path of the authentication file by setting the REGISTRY\_AUTH\_FILE
environment variable. `export REGISTRY_AUTH_FILE=path`

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) to connect to the registry.
The default certificates directory is _/etc/containers/certs.d_.

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--features**

Specify the features list which the list or index records as requirements for
the image.  This option is rarely used.

**--os**

Override the OS which the list or index records as a requirement for the image.
If *imageName* refers to a manifest list or image index, the OS information
will be retrieved from it.  Otherwise, it will be retrieved from the image's
configuration information.

**--os-features**

Specify the OS features list which the list or index records as requirements
for the image.  This option is rarely used.

**--os-version**

Specify the OS version which the list or index records as a requirement for the
image.  This option is rarely used.

**--tls-verify** *bool-value*

Require HTTPS and verification of certificates when talking to container registries (defaults to true).  TLS verification cannot be used when talking to an insecure registry.

**--variant**

Specify the variant which the list or index records for the image.  This option
is typically used to distinguish between multiple entries which share the same
architecture value, but which expect different versions of its instruction set.

## EXAMPLE

```
buildah manifest add mylist:v1.11 docker://fedora
506d8f4bb54931ea03a7e70173a0ed6302e3fb92dfadb3955ba5c17812e95c51: sha256:f81f09918379d5442d20dff82a298f29698197035e737f76e511d5af422cabd7
```

```
buildah manifest add --all mylist:v1.11 docker://fedora
506d8f4bb54931ea03a7e70173a0ed6302e3fb92dfadb3955ba5c17812e95c51: sha256:f81f09918379d5442d20dff82a298f29698197035e737f76e511d5af422cabd7
```

```
buildah manifest add --arch arm64 --variant v8 mylist:v1.11 docker://fedora@sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b
506d8f4bb54931ea03a7e70173a0ed6302e3fb92dfadb3955ba5c17812e95c51: sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b
```

## SEE ALSO
buildah(1), buildah-login(1), buildah-manifest(1), buildah-manifest-create(1), buildah-manifest-remove(1), buildah-manifest-annotate(1), buildah-manifest-inspect(1), buildah-manifest-push(1), buildah-rmi(1), docker-login(1)
