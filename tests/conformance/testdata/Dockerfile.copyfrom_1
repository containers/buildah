FROM busybox as base
RUN touch /a /b
FROM busybox
COPY --from=base /a /
RUN ls -al /a