FROM alpine as builder
RUN mkdir subdir
COPY hey .

FROM ubuntu
RUN  --mount=type=bind,source=.,dst=/tmp,z \
     --mount=type=tmpfs,dst=/var/tmp \
    cat /tmp/hey
RUN --mount=type=cache,from=builder,target=/cachedir cat /cachedir/hey
RUN --mount=type=secret,id=secret-foo,dst=secret1.txt cat secret1.txt
ARG TMP="/tmp"
ARG VARTMP="/var/tmp"
ARG CACHEDIR="/cachedir"
ARG TESTDIR="/testdir"
ARG SECRETFILE="secret1.txt"
RUN [ -d "/tmp" ] && echo "Directory $TMP exists."
RUN [ -d "/var/tmp" ] && echo "Directory $VARTMP exists."

# Following path should not exist after the --mount step
RUN [ ! -d "/testdir" ] && echo "Directory $TESTDIR DOES NOT exist."
RUN [ ! -d "/cachedir" ] && echo "Cache Directory $CACHEDIR DOES NOT exist."
RUN [ ! -f "secret1.txt" ] && echo "Secret File $SECRETFILE DOES NOT exist."
# This should fail
RUN cat /tmp/hey
