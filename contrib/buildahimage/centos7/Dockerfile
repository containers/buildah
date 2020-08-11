# centos7/Dockerfile
#
# Build a Buildah container image from the compatible
# version of Buildah on the Centos 7 System.
# This image can be used to create a secured container
# that runs safely with privileges within the container.
FROM centos:7

# Remove directories used by yum that are just taking
# up space.
RUN useradd build; yum -y update; yum -y reinstall shadow-utils; yum -y install buildah fuse-overlayfs xz; rm -rf /var/cache /var/log/dnf* /var/log/yum.*;

ADD https://raw.githubusercontent.com/containers/buildah/master/contrib/buildahimage/stable/containers.conf /etc/containers/

# Adjust storage.conf to enable Fuse storage.
RUN chmod 644 /etc/containers/containers.conf; sed -i -e '/size = ""/amount_program = "/usr/bin/fuse-overlayfs"' -e '/additionalimage.*/a "/var/lib/shared",' -e 's|^mountopt[[:space:]]*=.*$|mountopt = "nodev,fsync=0"|g' /etc/containers/storage.conf
RUN mkdir -p /var/lib/shared/overlay-images /var/lib/shared/overlay-layers /var/lib/shared/vfs-images /var/lib/shared/vfs-layers; touch /var/lib/shared/overlay-images/images.lock; touch /var/lib/shared/overlay-layers/layers.lock; touch /var/lib/shared/vfs-images/images.lock; touch /var/lib/shared/vfs-layers/layers.lock

# Set an environment variable to default to chroot isolation for RUN
# instructions and "buildah run".
ENV BUILDAH_ISOLATION=chroot
