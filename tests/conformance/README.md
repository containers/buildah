![buildah logo](https://cdn.rawgit.com/projectatomic/buildah/master/logos/buildah-logo_large.png)

# Buildah/Docker Conformance Test Suite

The conformance test for `buildah bud` is used to verify the images built with Buildah are equivalent to those built by  Docker. The test is impelemented with the [Ginkgo](https://github.com/onsi/ginkgo) BDD testing framework. The tests can be run with both the "ginkgo" and the "go test" commands.

## Installing dependencies

The dependencies for comformance testing include two parts:
* Binary required by the conformance test suite:
  * docker
  * podman
  * container-diff
* Binary required to run the tests:
  * ginkgo

### Install Buildah

Install Buildah using dnf, yum or apt-get, based on your distribution.  Buildah can also be cloned and installed from [GitHub](https://github.com/projectatomic/buildah/blob/master/install.md).
 
### Install Docker CE

Conformance tests use Docker CE to check the images built with Buildah. Install Docker CE with dnf, yum or apt-get, based on your distribution and verify that the Docker service is started as the default. In Fedora, RHEL and CentOS Docker rather than Docker CE may be installed by default, please verify that you install the correct variant of Docker.

### Install Podman

[Podman](https://github.com/containers/libpod) is used to push images built with Buildah to the docker deamon. It can be installed with dnf or yum in Fedora, RHEL and CentOS, it also can be installed from source code. If you want to install Podman from source code, please follow the [libpod Installation Instructions](https://github.com/containers/libpod/blob/master/install.md).

### Install container-diff

[container-diff](https://github.com/GoogleContainerTools/container-diff) is used for check images file system from Buildah and Docker. It can be installed with the following command:

```
curl -LO https://storage.googleapis.com/container-diff/latest/container-diff-linux-amd64 && chmod +x container-diff-linux-amd64 && mkdir -p $HOME/bin && export PATH=$PATH:$HOME/bin && mv container-diff-linux-amd64 $HOME/bin/container-diff
```

### Install ginkgo and gomega

We already have vendored ginkgo and gomega into the conformance tests, so if you want to just run the tests with "go test", you can skip this step.
Ginkgo is tested with Go 1.6+, please make sure your golang version meet the required version.
```
go get -u github.com/onsi/ginkgo/ginkgo  # installs the ginkgo CLI
go get -u github.com/onsi/gomega/...     # fetches the matcher library
export PATH=$PATH:$GOPATH/bin
```

## Run conformance tests

You can run the test with go test or ginkgo:
```
ginkgo -v  tests/conformance
```
or
```
go test -v ./tests/conformance
```

If you wan to run one of the test cases you can use flag "-focus":
```
ginkgo -v -focus "shell test" test/conformance
```
or
```
go test -c ./tests/conformance
./conformance.test -ginkgo.v -ginkgo.focus "shell test"
```

There are also some environment varibles that can be set during the test:

| Varible Name              | Useage  |
| :------------------------ | :-------------------------------------------------------- |
| BUILDAH\_BINARY | Used to set the Buildah binary path. Can be used for test installed rpm |
| TEST\_DATA\_DIR | Test data directory include the Dockerfiles and related files |
| DOCKER\_BINARY | Docker binary path. |
| BUILDAH\_$SUBCMD\_OPTIONS | Command line options for each Buildah command. $SUBCMD is the short command from "buildah -h". |
| $GLOBALOPTIONS | Global options from "buildah -h". The Varible Name is the option name which replace "-" with "\_" and with upper case |

Example to run conformance test for buildah bud with --format=docker:
```
Export BUILDAH_BUD_OPTIONS="--format docker"
ginkgo -v test/conformance
```
