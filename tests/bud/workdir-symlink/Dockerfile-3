# No directory created for the target of the symlink
FROM alpine
RUN ln -sf /var/lib/tempest /tempest
WORKDIR /tempest/lowerdir
RUN touch /etc/notareal.conf
RUN chmod 664 /etc/notareal.conf
RUN mkdir -p /tempest/lowerdir
COPY Dockerfile-3 ./Dockerfile-3
COPY Dockerfile-3 /tempest/Dockerfile-3
COPY Dockerfile-3 /tempest/lowerdir/Dockerfile-3
