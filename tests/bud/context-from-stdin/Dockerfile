FROM alpine as base
RUN echo "stdin-context" > /scratchfile

FROM scratch
COPY --from=base /scratchfile /
