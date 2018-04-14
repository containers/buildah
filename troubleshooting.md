![buildah logo](https://cdn.rawgit.com/projectatomic/buildah/master/logos/buildah-logo_large.png)

# Troubleshooting

## A list of common issues and solutions for Buildah

---
### No such image

When doing a `buildah pull` or `buildah bud` command and a "common" image can not be pulled,
it is likely that the `/etc/containers/registries.conf` file is either not installed or possibly
misconfigured.

#### Symptom
```
sudo buildah bud -f Dockerfile 
STEP 1: FROM alpine
error building: error creating build container: no such image "alpine" in registry: image not known
```

#### Solution

  * Verify that the `/etc/containers/registries.conf` file exists.  If not, verify that the skopeo-containers package is installed.
  * Verify that the entries in the `[registries.search]` section of the /etc/containers/registries file are valid and reachable.

---
### http: server gave HTTP response to HTTPS client

When doing a Buildah command such as `bud`, `commit`, `from`, or `push` to a registry,
tls verification is turned on by default.  If authentication is not used with
those commands, this error can occur.

#### Symptom
```
buildah push alpine docker://localhost:5000/myalpine:latest
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
