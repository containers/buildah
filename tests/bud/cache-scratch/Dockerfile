FROM alpine as build

FROM scratch
COPY --from=build / /
COPY --from=build / /
