FROM busybox as base
RUN mkdir -p /a && touch /a/1
FROM busybox
COPY --from=base a /a
RUN ls -al /a/1 && ! ls -al /a/a
