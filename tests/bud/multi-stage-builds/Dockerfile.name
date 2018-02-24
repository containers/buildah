FROM alpine as myname
COPY Dockerfile.name /

FROM scratch
COPY --from=myname /Dockerfile.name /Dockerfile.name
