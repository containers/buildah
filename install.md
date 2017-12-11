# Installation Instructions

Prior to installing Buildah, install the following packages on your linux distro:
* make
* golang (Requires version 1.8.1 or higher.)
* bats
* btrfs-progs-devel
* bzip2
* device-mapper-devel
* git
* go-md2man
* gpgme-devel
* glib2-devel
* libassuan-devel
* ostree-devel
* runc (Requires version 1.0 RC4 or higher.)
* skopeo-containers

## Fedora

In Fedora, you can use this command:

```
 dnf -y install \
    make \
    golang \
    bats \
    btrfs-progs-devel \
    device-mapper-devel \
    glib2-devel \
    gpgme-devel \
    libassuan-devel \
    ostree-devel \
    git \
    bzip2 \
    go-md2man \
    runc \
    skopeo-containers
```

Then to install Buildah on Fedora follow the steps in this example:


```
  mkdir ~/buildah
  cd ~/buildah
  export GOPATH=`pwd`
  git clone https://github.com/projectatomic/buildah ./src/github.com/projectatomic/buildah
  cd ./src/github.com/projectatomic/buildah
  make
  make install
  buildah --help
```

## RHEL, CentOS

In RHEL and CentOS 7, ensure that you are subscribed to `rhel-7-server-rpms`,
`rhel-7-server-extras-rpms`, and `rhel-7-server-optional-rpms`, then
run this command:

```
 yum -y install \
    make \
    golang \
    bats \
    btrfs-progs-devel \
    device-mapper-devel \
    glib2-devel \
    gpgme-devel \
    libassuan-devel \
    ostree-devel \
    git \
    bzip2 \
    go-md2man \
    runc \
    skopeo-containers
```

The build steps for Buildah on RHEL or CentOS are the same as Fedora, above.

## Ubuntu

In Ubuntu zesty and xenial, you can use this command:

```
  apt-get -y install software-properties-common
  add-apt-repository -y ppa:alexlarsson/flatpak
  add-apt-repository -y ppa:gophers/archive
  apt-add-repository -y ppa:projectatomic/ppa
  apt-get -y -qq update
  apt-get -y install bats btrfs-tools git libapparmor-dev libdevmapper-dev libglib2.0-dev libgpgme11-dev libostree-dev libseccomp-dev libselinux1-dev skopeo-containers go-md2man
  apt-get -y install golang-1.8
```
Then to install Buildah on Ubuntu follow the steps in this example:

```
  mkdir ~/buildah
  cd ~/buildah
  export GOPATH=`pwd`
  git clone https://github.com/projectatomic/buildah ./src/github.com/projectatomic/buildah
  cd ./src/github.com/projectatomic/buildah
  PATH=/usr/lib/go-1.8/bin:$PATH make runc all TAGS="apparmor seccomp"
  make install
  buildah --help
```
