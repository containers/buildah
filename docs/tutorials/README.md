![buildah logo](../../logos/buildah-logo_large.png)

# Buildah Tutorials

## Links to a number of useful tutorials for the Buildah project.

**[Introduction Tutorial](01-intro.md)**

Learn how to build container images compliant with the [Open Container Initiative](https://www.opencontainers.org/) (OCI) [image specification](https://github.com/opencontainers/image-spec) using Buildah.


**[Buildah and Registries Tutorial](02-registries-repositories.md)**

Learn how Buildah can be used to move OCI compliant images in and out of private or public registries.

**[Buildah ONBUILD Tutorial](03-on-build.md)**

Learn how Buildah can use the ONBUILD instruction in either a Dockerfile or via the `buildah config --onbuild` command to configure an image to run those instructions when the container is created.  In this manner you can front load setup of the container inside the image and minimalize the steps needed to create one or more containers that share a number of initial settings, but need a few differentiators between each.


