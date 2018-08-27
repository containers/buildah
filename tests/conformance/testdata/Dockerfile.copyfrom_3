FROM busybox as base
RUN touch /a
FROM busybox
WORKDIR /b
COPY --from=base /a .
RUN ls -al /b/a