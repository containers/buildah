# if TARGETARCH is defined, we'l pull the x86_64 image
FROM registry.fedoraproject.org/fedora-minimal:41${TARGETARCH:+-x86_64} AS amd64
# if TARGETARCH is defined, we'l pull the aarch64 image
FROM registry.fedoraproject.org/fedora-minimal:41${TARGETARCH:+-aarch64} AS arm64

# run "file" against both shared libraries
FROM registry.fedoraproject.org/fedora-minimal AS native
COPY --from=amd64 /lib64/libc.so.6 /libc-amd64
COPY --from=arm64 /lib64/libc.so.6 /libc-arm64
RUN microdnf -y install file && microdnf -y clean all
RUN file /libc-* | tee /libc-types.txt && touch -d @0 /libc-types.txt

# expect them to have different target architectures listed in their ELF headers
FROM registry.fedoraproject.org/fedora-minimal
COPY --from=native /libc-types.txt /
