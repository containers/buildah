# if TARGETARCH is defined, we'll pull the amd64 image from quay.io/libpod/busybox:glibc, otherwise we'll get the native one
FROM quay.io/libpod/busybox${TARGETARCH:+@sha256:32968e717e29f79e5b889721908b3be6d1573992e1f3ba4c9a41718e2728458c} AS amd64
# if TARGETARCH is defined, we'll pull the arm64 image from quay.io/libpod/busybox:glibc, otherwise we'll get the native one
FROM quay.io/libpod/busybox${TARGETARCH:+@sha256:1faaf7a75319417261267b3dcef6a2b14c8c5122d7fcc7abeb07a32bc19728fd} AS arm64

# run "readelf" against both shared libraries from any image that comes with it
FROM quay.io/fedora/python-311:latest AS native
USER 0:0
COPY --from=amd64 /lib64/libc.so.6 /libc-amd64
COPY --from=arm64 /lib64/libc.so.6 /libc-arm64
RUN readelf -h /libc-* | grep -i machine: | sort -u | tee /libc-types.txt && touch -d @0 /libc-types.txt

# expect to have read two different "Machine:" values from library headers
FROM scratch
COPY --from=native /libc-types.txt /
