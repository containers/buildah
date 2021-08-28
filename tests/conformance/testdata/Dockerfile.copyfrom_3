FROM busybox as base
RUN touch -t @1485449953 /a
FROM busybox
WORKDIR /b
COPY --from=base /a .
RUN ls -al /b/a