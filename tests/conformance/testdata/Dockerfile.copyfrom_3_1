FROM busybox as base
RUN touch /a
FROM busybox
WORKDIR /b
COPY --from=base /a ./b
RUN ls -al /b/b
