## buildah-mount "1" "March 2017" "buildah"

## NAME
buildah mount - Mount a working container's root filesystem.

## SYNOPSIS
**buildah** **mount**

**buildah** **mount** **containerID**

## DESCRIPTION
Mounts the specified container's root file system in a location which can be
accessed from the host, and returns its location.

If you execute the command without any arguments, the tool will list all of the
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

## SEE ALSO
buildah(1)
