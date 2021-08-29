# buildah-source-pull "1" "March 2021" "buildah"

## NAME
buildah\-source\-pull - Pull a source image from a registry to a specified path

## SYNOPSIS
**buildah source pull** [*options*] *registry* *path*

## DESCRIPTION
Pull a source image from a registry to a specified path.  The pull operation
will fail if the image does not comply with a source-image OCI rartifact.

Note that the buildah-source command and all its subcommands are experimental
and may be subject to future changes.

## OPTIONS

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--tls-verify** *bool-value*

Require HTTPS and verification of certificates when talking to container
registries (defaults to true).  TLS verification cannot be used when talking to
an insecure registry.
