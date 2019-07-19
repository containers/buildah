# buildah-unshare "1" "June 2018" "buildah"

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

buildah unshare is useful for troubleshooting unprivileged operations and for
manually clearing storage and other data related to images and containers.

It is also useful if you want to use the `buildah mount` command.  If an unprivileged users wants to mount and work with a container, then they need to execute
buildah unshare.  Executing `buildah mount` fails for unprivileged users unless the user is running inside a `buildah unshare` session.

## OPTIONS
**--mount** [*VARIABLE=]containerNameOrID*

Mount the *containerNameOrID* container while running *command*, and set the
environment variable *VARIABLE* to the path of the mountpoint.  If *VARIABLE*
is not specified, it defaults to *containerNameOrID*, which may not be a valid
name for an environment variable.

## EXAMPLE

buildah unshare id

buildah unshare pwd

buildah unshare cat /proc/self/uid\_map /proc/self/gid\_map

buildah unshare rm -fr $HOME/.local/share/containers/storage /var/run/user/\`id -u\`/run

buildah unshare --mount containerID sh -c 'cat ${containerID}/etc/os-release'

If you want to use buildah with a mount command then you can create a script that looks something like:

```
cat buildah-script.sh << _EOF
ctr=$(buildah from scratch)
mnt=$(buildah mount $ctr)
dnf -y install --installroot=$mnt PACKAGES
dnf -y clean all --installroot=$mnt
buildah config --entrypoint="/bin/PACKAGE" --env "FOO=BAR" $ctr
buildah commit $ctr IMAGENAME
buildah unmount $ctr
_EOF
```
Then execute it with:
```
buildah unshare buildah-script.sh
```

## SEE ALSO
buildah(1), buildah-mount(1), namespaces(7), newuidmap(1), newgidmap(1), user\_namespaces(7)
