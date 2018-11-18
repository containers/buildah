FROM alpine AS uuid
COPY uuid /src

FROM alpine AS date
COPY date /src

FROM alpine
COPY --from=uuid /src/data /uuid
COPY --from=date /src/data /date
