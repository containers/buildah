FROM busybox as base
RUN mkdir -p /a && touch -t @1485449953 /a/1
FROM busybox
COPY --from=base /a/1 /a/b/c
RUN ls -al /a/b/c && ! ls -al /a/b/1