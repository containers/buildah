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
