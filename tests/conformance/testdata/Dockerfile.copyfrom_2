FROM busybox as base
RUN touch -t @1485449953 /a
FROM busybox
COPY --from=base /a /a
RUN ls -al /a