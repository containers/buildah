FROM busybox as base
RUN mkdir -p /a && touch /a/b
FROM busybox
COPY --from=base /a/b /a
RUN ls -al /a && ! ls -al /b