

# Library of packages required at runtime kept on it's own so that
# the libpod project may also consume it during VM image build-time.


UBUNTU_BUILDAH_PACKAGES=( \
    aufs-tools
    bats
    bzip2
    coreutils
    curl
    git
    go-md2man
    golang
    libdevmapper-dev
    libglib2.0-dev
    libgpgme11-dev
    libseccomp-dev
    libselinux-dev
    openssl
    podman
    jq
    netcat
    rsync
    runc
    scons
    vim
    wget
    yum-utils
    zlib1g-dev
    xz-utils
)

FEDORA_BUILDAH_PACKAGES=(\
    bats
    btrfs-progs-devel
    bzip2
    containers-common
    device-mapper-devel
    findutils
    git
    glib2-devel
    glibc-static
    gnupg
    go-md2man
    golang
    gpgme-devel
    libassuan-devel
    libseccomp-devel
    make
    nmap-ncat
    ostree-devel
    podman
    jq
    rsync
    runc
    skopeo-containers
    wget
    xz
)
