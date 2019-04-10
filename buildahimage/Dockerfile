FROM fedora
RUN yum -y install buildah --exclude container-selinux --enablerepo updates-testing; rm -rf /var/cache /var/log/dnf* /var/log/yum.*
RUN sed 's|^#mount_program|mount_program|g' -i /etc/containers/storage.conf
ENV _BUILDAH_STARTED_IN_USERNS="" BUILDAH_ISOLATION=chroot
COPY Dockerfile /root

