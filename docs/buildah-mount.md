# buildah-mount "1" "March 2017" "buildah"

## NAME
buildah\-mount - Mount a working container's root filesystem.

## SYNOPSIS
**buildah mount** [*container* ...]

## DESCRIPTION
Mounts the specified container's root file system in a location which can be
accessed from the host, and returns its location.

If the mount command is invoked without any arguments, the tool will list all of the
currently mounted containers.

## RETURN VALUE
The location of the mounted file system.  On error an empty string and errno is
returned.

## OPTIONS

**--notruncate**

Do not truncate IDs in output.

## EXAMPLE

buildah mount c831414b10a3

/var/lib/containers/storage/overlay2/f3ac502d97b5681989dff84dfedc8354239bcecbdc2692f9a639f4e080a02364/merged

buildah mount

c831414b10a3 /var/lib/containers/storage/overlay2/f3ac502d97b5681989dff84dfedc8354239bcecbdc2692f9a639f4e080a02364/merged

a7060253093b /var/lib/containers/storage/overlay2/0ff7d7ca68bed1ace424f9df154d2dd7b5a125c19d887f17653cbcd5b6e30ba1/merged

buildah mount efdb54a2f0d7 644db0db094c adffbea87fa8

efdb54a2f0d7 /var/lib/containers/storage/overlay/f8cac5cce73e5102ab321cc5b57c0824035b5cb82b6822e3c86ebaff69fefa9c/merged
644db0db094c /var/lib/containers/storage/overlay/c3ec418be5bda5b72dca74c4d397e05829fe62ecd577dd7518b5f7fc1ca5f491/merged
adffbea87fa8 /var/lib/containers/storage/overlay/03a071f206f70f4fcae5379bd5126be86b5352dc2a0c3449cd6fca01b77ea868/merged

## SEE ALSO
buildah(1)
