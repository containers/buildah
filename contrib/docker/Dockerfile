FROM fedora
RUN dnf -y update && dnf -y clean all
RUN dnf -y install btrfs-progs-devel containers-common device-mapper-devel golang go-md2man gpgme-devel libassuan-devel libseccomp-devel make net-tools ostree-devel runc shadow-utils glibc-static libselinux-static libseccomp-static && dnf -y clean all
COPY . /go/src/github.com/containers/buildah
RUN env GOPATH=/go make -C /go/src/github.com/containers/buildah clean all install
RUN sed -i -r -e 's,driver = ".*",driver = "vfs",g' /etc/containers/storage.conf
ENV BUILDAH_ISOLATION chroot
WORKDIR /root
CMD /bin/bash
