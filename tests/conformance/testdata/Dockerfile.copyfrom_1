FROM busybox as base
RUN touch -t @1485449953 /a /b
FROM busybox
COPY --from=base /a /
RUN ls -al /a