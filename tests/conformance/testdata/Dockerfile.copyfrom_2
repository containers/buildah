FROM busybox as base
RUN touch /a
FROM busybox
COPY --from=base /a /a
RUN ls -al /a