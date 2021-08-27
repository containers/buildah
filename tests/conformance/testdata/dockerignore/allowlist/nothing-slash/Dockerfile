FROM busybox
RUN touch -t @1485449953 /file
FROM scratch
COPY --from=0 /file /
ADD / /
