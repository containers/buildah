# buildah-logout "1" "Apr 2019" "buildah"

## NAME
buildah\-logout - Logout of a container registry

## SYNOPSIS
**buildah logout** [*options*] *registry*

## DESCRIPTION
**buildah logout** logs out of a specified registry server by deleting the cached credentials
stored in the **auth.json** file. The path of the authentication file can be overridden by the user by setting the **authfile** flag.
The default path used is **${XDG\_RUNTIME_DIR}/containers/auth.json**.  See containers-auth.json(5) for more information.
All the cached credentials can be removed by setting the **all** flag.

**buildah [GLOBAL OPTIONS]**

**buildah logout [GLOBAL OPTIONS]**

**buildah logout [OPTIONS] REGISTRY [GLOBAL OPTIONS]**

## OPTIONS

**--all**, **-a**

Remove the cached credentials for all registries in the auth file

**--authfile**

Path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json.  See containers-auth.json(5) for more information.

Note: You can also override the default path of the authentication file by setting the REGISTRY\_AUTH\_FILE
environment variable. `export REGISTRY_AUTH_FILE=path`

**--compat-auth-file**=*path*

Instead of updating the default credentials file, update the one at *path*, and use a Docker-compatible format.

**--help**, **-h**

Print usage statement

## EXAMPLES

```
$ buildah logout quay.io
Removed login credentials for quay.io
```

```
$ buildah logout --authfile authdir/myauths.json quay.io
Removed login credentials for quay.io
```

```
$ buildah logout --all
Remove login credentials for all registries
```

## SEE ALSO
buildah(1), buildah-login(1), containers-auth.json(5)
