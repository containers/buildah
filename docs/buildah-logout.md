# buildah-logout "1" "Apr 2019" "buildah"

## NAME
buildah\-logout - Logout of a container registry

## SYNOPSIS
**buildah logout** [*options*] *registry*

## DESCRIPTION
**buildah logout** logs out of a specified registry server by deleting the cached credentials stored in the kernel keyring.
If the system does not support kernel keyring or the authorization state is not found there, Buildah will check the authentication file.
The path of the authentication file can be overridden by the user by setting the **authfile** option.
The default path used is **${XDG\_RUNTIME_DIR}/containers/auth.json**.
All the authentication file cached credentials can be removed by setting the **all** option.

**buildah [GLOBAL OPTIONS]**

**bildah logout [GLOBAL OPTIONS]**

**buildah logout [OPTIONS] REGISTRY [GLOBAL OPTIONS]**

## OPTIONS

**--authfile**

Path of the authentication file. By default, the authentication storage is the kernel keyring. If the system does not support kernel keyring, Buildah will use the authentication file.Default is ${XDG\_RUNTIME\_DIR}/containers/auth.json, which is set using `buildah login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

Note: The default path of the authentication file can also be overridden by setting the REGISTRY_AUTH_FILE environment variable. `export REGISTRY_AUTH_FILE=path`

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