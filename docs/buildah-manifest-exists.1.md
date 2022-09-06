% buildah-manifest-exists(1)

## NAME
buildah\-manifest\-exists - Check if the given manifest list exists in local storage

## SYNOPSIS
**buildah manifest exists** *manifest*

## DESCRIPTION
**buildah manifest exists** checks if a manifest list exists in local storage. Buildah will
return an exit code of `0` when the manifest list is found. A `1` will be returned otherwise.
An exit code of `125` indicates there was another issue.


## OPTIONS

#### **--help**, **-h**

Print usage statement.

## EXAMPLE

Check if a manifest list called `list1` exists (the manifest list does actually exist).
```
$ buildah manifest exists list1
$ echo $?
0
$
```

Check if an manifest called `mylist` exists (the manifest list does not actually exist).
```
$ buildah manifest exists mylist
$ echo $?
1
$
```

## SEE ALSO
**[buildah(1)](buildah.1.md)**, **[buildah-manifest(1)](buildah-manifest.1.md)**
