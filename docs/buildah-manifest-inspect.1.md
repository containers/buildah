# buildah-manifest-inspect "1" "September 2019" "buildah"

## NAME

buildah\-manifest\-inspect - Display a manifest list or image index.

## SYNOPSIS

**buildah manifest inspect** *listNameOrIndexName*

## DESCRIPTION

Displays the manifest list or image index stored using the specified image name.

## RETURN VALUE

A formatted JSON representation of the manifest list or image index.

## OPTIONS

**--authfile** *path*

Path of the authentication file. Default is ${XDG\_RUNTIME\_DIR}/containers/auth.json, which is set using `buildah login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

**--tls-details** *path*

Path to a `containers-tls-details.yaml(5)` file.

If not set, defaults to a reasonable default that may change over time (depending on system's global policy,
version of the program, version of the Go language, and the like).

Users should generally not use this option unless they have a process to ensure that the configuration will be kept up to date.

**--tls-verify** *bool-value*

Require HTTPS and verification of certificates when talking to container registries (defaults to true).  TLS verification cannot be used when talking to an insecure registry.

## EXAMPLE

```
buildah manifest inspect mylist:v1.11
```

## SEE ALSO
buildah(1), buildah-manifest(1), buildah-manifest-create(1), buildah-manifest-add(1), buildah-manifest-remove(1), buildah-manifest-annotate(1), buildah-manifest-push(1), buildah-rmi(1)
