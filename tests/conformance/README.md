![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

# Buildah/Docker Conformance Test Suite

The conformance test for buildah is used to verify the images built with Buildah are equivalent to those built by Docker.  It does this by building an image using the version of buildah library that's being tested, building what should be the same image using the docker engine's build API, and comparing them.

## Installing dependencies

The additional dependencies for conformance testing are:
  * docker

### Install Docker CE

Conformance tests use Docker CE to build images to be compared with images built with Buildah.  Install Docker CE with dnf, yum or apt-get, based on your distribution and verify that the `docker` service is started.  In Fedora, RHEL and CentOS `docker` or `moby-engine` rather than Docker CE may be installed by default.  In Debian or Ubuntu you may instead have the `docker.io` package.  Please verify that you install at least version 19.03.

## Run conformance tests

You can run all of the tests with go test:
```
go test -v -tags "$(./btrfs_tag.sh) $(./btrfs_installed_tag.sh) $(./libdm_tag.sh)" ./tests/conformance
```

If you want to run one of the test cases you can use the "-run" flag:
```
go test -v -tags "$(./btrfs_tag.sh) $(./btrfs_installed_tag.sh) $(./libdm_tag.sh)" -run TestConformance/shell ./tests/conformance
```

If you also want to build and compare on a line-by-line basis, run:
```
go test -v -timeout=60m -tags "$(./btrfs_tag.sh) $(./btrfs_installed_tag.sh) $(./libdm_tag.sh)" ./tests/conformance -compare-layers
```
