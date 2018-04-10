FROM scratch as myname
COPY Dockerfile.name /

FROM scratch as myname2
COPY Dockerfile.index /

FROM scratch
COPY Dockerfile.mixed /

FROM scratch
COPY --from=myname /Dockerfile.name /Dockerfile.name
COPY --from=1 /Dockerfile.index /Dockerfile.index
COPY --from=2 /Dockerfile.mixed /Dockerfile.mixed
