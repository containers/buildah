![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

# Buildah Tutorial 6
## Using Buildah in CI

This tutorial will walk you through setting up GitLab CI with Buildah

The instructions have been tested with Buildah 1.30.0 and GitLab Runner 16.1.0 (running in Podman).

Note that VFS is used for storage instead of the more performant fuse-overlayfs or overlayfs. But the the latter do not work at the moment in unprivileged environments.

### Create a project

Let's make a simple project that we can build using Buildah. This code will create a git repo and a small program that simply reads a file named `resource.txt`. We also create a `Dockerfile` to build an image for it. The `Dockerfile` has a simulated build step, which delays five seconds to give the impression of a slow build process.

```
git init
cat <<EOF >main.sh
#!/bin/sh
echo Welcome to the built image!
cat resource.txt
EOF
echo "Hello world!" > resource.txt
chmod +x main.sh
cat <<EOF >Dockerfile
FROM registry.fedoraproject.org/fedora:latest
COPY main.sh /
RUN echo Building image... && sleep 5 && echo Built image
COPY resource.txt /
ENTRYPOINT ["main.sh"]
EOF
```

Locally, we can build this image easily using `buildah bud .`

However, we want to use Continuous Integration to build this image every time a change is made.

### Setting up CI

Let's make a `.gitlab-ci.yml` file with the following content:

```
variables:
  GIT_SUBMODULE_STRATEGY: recursive # clone submodules
  BUILDAH_LAYERS: "true" # store intermediate layers so we can cache them to speed up subsequent builds
  BUILDAH_ISOLATION: "chroot" # needed due to restricted unprivileged environment (but less isolated than other modes)
  STORAGE_DRIVER: "vfs" # needed due to restricted unprivileged environment (but slower than other drivers)

build:
  stage: build
  image: quay.io/buildah/stable # pull down the Buildah image
  cache:
    # This block allows us to cache the intermediate layers from the build
    key: cache
    paths:
      - .cache/
  script:
    # passing --root causes Buildah to place intermediate images in the cache directory where GitLab can cache it
    # build the image and tag it
    - buildah --root=.cache bud -t "${CI_REGISTRY_IMAGE}:${CI_COMMIT_SHA}" .
    # push the image to the GitLab container registry
    - buildah --root=.cache push "${CI_REGISTRY_IMAGE}:${CI_COMMIT_SHA}"
    # push the image under an alias of the current branch/tag name, too
    - buildah --root=.cache push "${CI_REGISTRY_IMAGE}:${CI_COMMIT_SHA}" "${CI_REGISTRY_IMAGE}:${CI_COMMIT_REF_NAME}"
```

You can add more steps to the yaml file to pull this built image down and run tests with it, or you can even deploy it straight to your staging environment. 