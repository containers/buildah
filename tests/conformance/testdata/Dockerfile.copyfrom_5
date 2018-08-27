FROM busybox as base
RUN mkdir -p /a/b && touch /a/b/1 /a/b/2
FROM busybox
COPY --from=base /a/b/* /b/
RUN ls -al /b/1 /b/2 /b && ! ls -al /a /b/a /b/b
