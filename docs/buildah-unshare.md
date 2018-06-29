# buildah-unshare "19" "June 2018" "buildah"

## NAME
buildah\-unshare - Run a command inside of a modified user namespace.

## SYNOPSIS
**buildah unshare** [*options*] [**--**] [*command*]

## DESCRIPTION
Launches a process (by default, *$SHELL*) in a new user namespace.  The user
namespace is configured so that the invoking user's UID and primary GID appear
to be UID 0 and GID 0, respectively.  Any ranges which match that user and
group in /etc/subuid and /etc/subgid are also mapped in as themselves with the
help of the *newuidmap(1)* and *newgidmap(1)* helpers.

This is mainly useful for troubleshooting unprivileged operations and for
manually clearing storage and other data related to images and containers.

## EXAMPLE

buildah unshare id

buildah unshare pwd

buildah unshare cat /proc/self/uid\_map

buildah unshare cat /proc/self/gid\_map

buildah unshare rm -fr $HOME/.local/share/containers/storage /var/run/user/\`id -u\`/run

## SEE ALSO
buildah(1), namespaces(7), newuidmap(1), newgidmap(1), user\_namespaces(7)
