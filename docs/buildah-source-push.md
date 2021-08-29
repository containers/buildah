# buildah-source-push "1" "March 2021" "buildah"

## NAME
buildah\-source\-push - Push a source image from a specified path to a registry.

## SYNOPSIS
**buildah source push** [*options*] *path* *registry*

## DESCRIPTION
Push a source image from a specified path to a registry.

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
