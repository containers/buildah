FROM busybox as base
RUN touch /b
FROM busybox
COPY --from=base /b /a
RUN ls -al /a && ! ls -al /b