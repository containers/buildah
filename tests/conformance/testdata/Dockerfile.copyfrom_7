FROM busybox as base
RUN touch -t @1485449953 /b
FROM busybox
COPY --from=base /b /a
RUN ls -al /a && ! ls -al /b