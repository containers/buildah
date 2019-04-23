# git/Dockerfile
#
# Build a Buildah container image from the latest
# upstream version of Buildah on GitHub.
# https://github.com/containers/buildah
# This image can be used to create a secured container
# that runs safely with privileges within the container.
# The containers created by this image also come with a
# Buildah development environment in /root/buildah.
#
FROM fedora
ENV GOPATH=/root/buildah

# Install the software required to build Buildah.
RUN dnf -y install --enablerepo=updates-testing --exclude=container-selinux \
     make \
     golang \
     bats \
     btrfs-progs-devel \
     device-mapper-devel \
     glib2-devel \
     gpgme-devel \
     libassuan-devel \
     libseccomp-devel \
     ostree-devel \
     git \
     bzip2 \
     go-md2man \
     runc \
     fuse-overlayfs \
     fuse3 \
     containers-common;

# Create a directory and clone from the Buildah
# GitHub repository, then make and install Buildah
# to the container.
RUN mkdir /root/buildah; \
     git clone https://github.com/containers/buildah /root/buildah/src/github.com/containers/buildah; \
     cd /root/buildah/src/github.com/containers/buildah; \
     make;\
     make install

# Adjust storage.conf to enable Fuse storage.
RUN sed -i -e 's|^#mount_program|mount_program|g' -e '/additionalimage.*/a "/var/lib/shared",' /etc/containers/storage.conf
RUN mkdir -p /var/lib/shared/overlay-images /var/lib/shared/overlay-layers; touch /var/lib/shared/overlay-images/images.lock; touch /var/lib/shared/overlay-layers/layers.lock

# This needs to be fixed in buildah and container/storage, but for now
# this hack works.
RUN mkdir /root/.config && ln -s /etc/containers /root/.config/containers

# Set up environment variables to note that this is
# not starting with usernamespace and default to 
# isolate the filesystem with chroot.
ENV _BUILDAH_STARTED_IN_USERNS="" BUILDAH_ISOLATION=chroot

