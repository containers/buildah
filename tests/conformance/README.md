# Conformance tests

The conformance test for buildah bud is used to check the images build with buildah against docker. The test is impelement with [Ginkgo](https://github.com/onsi/ginkgo) BDD testing framework. Can be run with both ginkgo and go test.

## Installing dependencies

The dependencies for comformance test include two part except the binarys request by buildah:
* Binary request by test case
  * docker
  * podman
  * container-diff
* Binary to run the tests
  * ginkgo

### Install docker

Conformance tests use docker to check the images build with buildah. So we need install docker with dnf, yum or apt-get based on your distribution and need make sure to start docker service as default. In Fedora, RHEL and CentOS

The following instructions use $GOPATH as your GOPATH. You can export GOPATH to your env then run them directly.

### Install podman

[Podman](https://github.com/containers/libpod) is used for push images build with buildah to docker deamon. It can be installed with dnf or yum in Fedora, RHEL and CentOS, also can be installed from source code. If you want to install podman from source code, please following the [libpod Installation Instructions](https://github.com/containers/libpod/blob/master/install.md)

### Install container-diff

[container-diff](https://github.com/GoogleContainerTools/container-diff) is used for check images file system from buildah and docker. It can be installed with following command:

```
curl -LO https://storage.googleapis.com/container-diff/latest/container-diff-linux-amd64 && chmod +x container-diff-linux-amd64 && mkdir -p $HOME/bin && export PATH=$PATH:$HOME/bin && mv container-diff-linux-amd64 $HOME/bin/container-diff
```
### Install ginkgo and gomega

As we already add ginkgo and gomega to our vendor, so if you want to just run the tests with "go test", you can skip this step.
Ginkgo is tested with Go 1.6+, please make sure your golang version meet the request.
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

There are also some env varibles can be set during the test:

| Varible Name              | Useage  |
| :------------------------ | :-------------------------------------------------------- |
| BUILDAH\_BINARY | Used to set builah binary path. Can be used for test installed rpm |
| TEST\_DATA\_DIR | Test data directory include the Dockerfiles and related files |
| DOCKER\_BINARY | Docker binary path. |
| BUILDAH\_$SUBCMD\_OPTIONS | Command line options for each buildah command. $SUBCMD is the short command from "buildah -h". |
| $GLOBALOPTIONS | Global options from "buildah -h". The Varible Name is the option name which replace "-" with "\_" and with upper case |

Examples for run conformance test for buildah bud with --format=docker:
```
Export BUILDAH_BUD_OPTIONS="--format docker"
ginkgo -v test/conformance
```
