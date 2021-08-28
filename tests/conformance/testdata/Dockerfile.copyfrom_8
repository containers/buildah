FROM busybox as base
RUN mkdir -p /a && touch -t @1485449953 /a/b
FROM busybox
COPY --from=base /a/b /a
RUN ls -al /a && ! ls -al /b