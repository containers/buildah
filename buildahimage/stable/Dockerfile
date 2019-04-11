# stable/Dockerfile
#
# Build a Buildah container image from the latest
# stable version of Buildah on the Fedoras Updates System.
# https://bodhi.fedoraproject.org/updates/?search=buildah
# This image can be used to create a secured container
# that runs safely with privileges within the container.
#
FROM fedora

# Don't include container-selinux and remove 
# directories used by dnf that are just taking
# up space.
RUN yum -y install buildah --exclude container-selinux; rm -rf /var/cache /var/log/dnf* /var/log/yum.*

# Adjust storage.conf to enable Fuse storage.
RUN sed 's|^#mount_program|mount_program|g' -i /etc/containers/storage.conf

# Set up environment variables to note that this is
# not starting with usernamespace and default to 
# isolate the filesystem with chroot.
ENV _BUILDAH_STARTED_IN_USERNS="" BUILDAH_ISOLATION=chroot
