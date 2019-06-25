# buildah-login "1" "Apr 2019" "buildah"

## NAME
buildah\-login - Login to a container registry

## SYNOPSIS
**buildah login** [*options*] *registry*

## DESCRIPTION
**buildah login** logs into a specified registry server with the correct username
and password. **buildah login** reads in the username and password from STDIN.
The username and password can also be set using the **username** and **password** flags.
The path of the authentication file can be specified by the user by setting the **authfile**
flag. The default path used is **${XDG\_RUNTIME_DIR}/containers/auth.json**.

**buildah [GLOBAL OPTIONS]**

**buildah login [GLOBAL OPTIONS]**

**buildah login [OPTIONS] REGISTRY [GLOBAL OPTIONS]**

## OPTIONS

**--password, -p**

Password for registry

**--password-stdin**

Take the password from stdin

**--username, -u**

Username for registry

**--authfile**

Path of the authentication file. Default is ${XDG_\RUNTIME\_DIR}/containers/auth.json

Note: You can also override the default path of the authentication file by setting the REGISTRY\_AUTH\_FILE
environment variable. `export REGISTRY_AUTH_FILE=path`

**--get-login**

Return the logged-in user for the registry.  Return error if no login is found.

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) to connect to the registry.
Default certificates directory is _/etc/containers/certs.d_.

**--tls-verify**

Require HTTPS and verify certificates when contacting registries (default: true). If explicitly set to true,
then TLS verification will be used. If set to false, then TLS verification will not be used. If not specified,
TLS verification will be used unless the target registry is listed as an insecure registry in registries.conf.

**--help**, **-h**

Print usage statement

## EXAMPLES

```
$ buildah login docker.io
Username: qiwanredhat
Password:
Login Succeeded!
```

```
$ buildah login -u testuser -p testpassword localhost:5000
Login Succeeded!
```

```
$ buildah login --authfile ./auth.json docker.io
Username: qiwanredhat
Password:
Login Succeeded!
```

```
$ buildah login --tls-verify=false -u test -p test localhost:5000
Login Succeeded!
```

```
$ buildah login --cert-dir /etc/containers/certs.d/ -u foo -p bar localhost:5000
Login Succeeded!
```

```
$ buildah login -u testuser  --password-stdin < pw.txt docker.io
Login Succeeded!
```

```
$ echo $testpassword | buildah login -u testuser --password-stdin docker.io
Login Succeeded!
```

## SEE ALSO
buildah(1), buildah-logout(1)
