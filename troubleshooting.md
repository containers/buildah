![buildah logo](https://cdn.rawgit.com/projectatomic/buildah/master/logos/buildah-logo_large.png)

# Troubleshooting

## A list of common issues and solutions for Buildah

---
### 1) No such image

When doing a `buildah pull` or `buildah bud` command and a "common" image can not be pulled,
it is likely that the `/etc/containers/registries.conf` file is either not installed or possibly
misconfigured.

#### Symptom
```console
$ sudo buildah bud -f Dockerfile 
STEP 1: FROM alpine
error building: error creating build container: no such image "alpine" in registry: image not known
```

#### Solution

  * Verify that the `/etc/containers/registries.conf` file exists.  If not, verify that the containers-common package is installed.
  * Verify that the entries in the `[registries.search]` section of the /etc/containers/registries file are valid and reachable.

---
### 2) http: server gave HTTP response to HTTPS client

When doing a Buildah command such as `bud`, `commit`, `from`, or `push` to a registry,
tls verification is turned on by default.  If authentication is not used with
those commands, this error can occur.

#### Symptom
```console
# buildah push alpine docker://localhost:5000/myalpine:latest
Getting image source signatures
Get https://localhost:5000/v2/: http: server gave HTTP response to HTTPS client
```

#### Solution

By default tls verification is turned on when communicating to registries from
Buildah.  If the registry does not require authentication the Buildah commands
such as `bud`, `commit`, `from` and `pull` will fail unless tls verification is turned
off using the `--tls-verify` option.  **NOTE:** It is not at all recommended to
communicate with a registry and not use tls verification.

  * Turn off tls verification by passing false to the tls-verification option.
  * I.e. `buildah push --tls-verify=false alpine docker://localhost:5000/myalpine:latest`

---
### 3) `buildah run` command fails with pipe or output redirection

When doing a `buildah run` command while using a pipe ('|') or output redirection ('>>'),
the command will fail, often times with a `command not found` type of error.

#### Symptom
When executing a `buildah run` command with a pipe or output redirection such as the
following commands:

```console
# buildah run $whalecontainer /usr/games/fortune -a | cowsay
# buildah run $newcontainer echo "daemon off;" >> /etc/nginx/nginx.conf
# buildah run $newcontainer echo "nginx on Fedora" > /usr/share/nginx/html/index.html
```
the `buildah run` command will not complete and an error will be raised.

#### Solution
There are two solutions to this problem.  The
[`podman run`](https://github.com/containers/libpod/blob/master/docs/podman-run.1.md)
command can be used in place of `buildah run`.  To still use `buildah run`, surround
the command with single quotes and use `bash -c`.  The previous examples would be
changed to:

```console
# buildah run bash -c '$whalecontainer /usr/games/fortune -a | cowsay'
# buildah run bash -c '$newcontainer echo "daemon off;" >> /etc/nginx/nginx.conf'
# buildah run bash -c '$newcontainer echo "nginx on Fedora" > /usr/share/nginx/html/index.html'
```

---
### 4) `buildah push alpine oci:~/myalpine:latest` fails with lstat error

When doing a `buildah push` command and the target image has a tilde (`~`) character
in it, an lstat error will be raised stating there is no such file or directory.
This is expected behavior for shell expansion of the tilde character as it is only
expanded at the start of a word.  This behavior is documented
[here](https://www.gnu.org/software/libc/manual/html_node/Tilde-Expansion.html).

#### Symptom
```console
$ sudo pull alpine
$ sudo buildah push alpine oci:~/myalpine:latest
lstat /home/myusername/~: no such file or directory
```

#### Solution

  * Replace `~` with `$HOME` or the fully specified directory `/home/myusername`.
    * `$ sudo buildah push alpine oci:${HOME}/myalpine:latest`
---
