FROM scratch
COPY Dockerfile.index /

FROM alpine
COPY --from=0 /Dockerfile.index /Dockerfile.index
