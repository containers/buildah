FROM busybox AS base
FROM scratch
COPY --from=base /bin/ls /test/
COPY --from=base /bin/sh /bin/
RUN /test/ls -lR /test/ls
