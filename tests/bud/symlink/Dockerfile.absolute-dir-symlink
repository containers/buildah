FROM ubuntu as builder
RUN mkdir -p /my/data && touch /my/data/myexe && ln -s /my/data /data

FROM ubuntu
COPY --from=builder /data /data
VOLUME [ "/data" ]
