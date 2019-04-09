# buildah-logout "1" "Apr 2019" "buildah"

## NAME
buildah\-logout - Logout of a container registry

## SYNOPSIS
**buildah logout** [*options*] *registry*

## DESCRIPTION
**buildah logout** logs out of a specified registry server by deleting the cached credentials
stored in the **auth.json** file. The path of the authentication file can be overridden by the user by setting the **authfile** flag.
The default path used is **${XDG\_RUNTIME_DIR}/containers/auth.json**.
All the cached credentials can be removed by setting the **all** flag.

**buildah [GLOBAL OPTIONS]**

**bildah logout [GLOBAL OPTIONS]**

**buildah logout [OPTIONS] REGISTRY [GLOBAL OPTIONS]**

## OPTIONS

**--authfile**

Path of the authentication file. Default is ${XDG_\RUNTIME\_DIR}/containers/auth.json

Note: You can also override the default path of the authentication file by setting the REGISTRY\_AUTH\_FILE
environment variable. `export REGISTRY_AUTH_FILE=path`

**--all, -a**

Remove the cached credentials for all registries in the auth file

**--help**, **-h**

Print usage statement

## EXAMPLES

```
$ buildah logout docker.io
Removed login credentials for docker.io
```

```
$ bildah logout --authfile authdir/myauths.json docker.io
Removed login credentials for docker.io
```

```
$ buildah logout --all
Remove login credentials for all registries
```

## SEE ALSO
buildah(1), buildah-login(1)