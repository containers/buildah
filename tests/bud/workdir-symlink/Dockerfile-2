# No directory created for the target of the symlink
FROM alpine
RUN ln -sf /var/lib/tempest /tempest
WORKDIR /tempest
RUN touch /etc/notareal.conf
RUN chmod 664 /etc/notareal.conf
COPY Dockerfile-2 ./Dockerfile-2
