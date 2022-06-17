FROM ubuntu:latest
RUN touch /1
RUN touch hello

FROM alpine:latest AS mytarget
RUN touch /2
# Just add a copy so we don't skip stage:0
COPY --from=0 hello .

FROM busybox:latest AS mytarget2 
RUN touch /3
# Just add a copy so we don't skip stage:1
COPY --from=1 hello .
