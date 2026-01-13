# Pull architecture-specific ubuntu images from cached registry
FROM --platform=linux/amd64 quay.io/libpod/ubuntu:latest AS amd64
FROM --platform=linux/arm64 quay.io/libpod/ubuntu:latest AS arm64

# run "file" against both shared libraries
FROM quay.io/libpod/ubuntu:latest AS native
COPY --from=amd64 /lib/x86_64-linux-gnu/libc.so.6 /libc-amd64
COPY --from=arm64 /lib/aarch64-linux-gnu/libc.so.6 /libc-arm64
RUN apt-get update && apt-get install -y file && apt-get clean
RUN file /libc-* | tee /libc-types.txt && touch -d @0 /libc-types.txt

# expect them to have different target architectures listed in their ELF headers
FROM quay.io/libpod/ubuntu:latest
COPY --from=native /libc-types.txt /
