FROM alpine
RUN adduser -D -H testuser && addgroup testgroup
RUN mkdir -p /vol/subvol
RUN chown testuser:testgroup /vol/subvol
VOLUME /vol/subvol

# Run some command after VOLUME to ensure that the volume cache behavior is invoked
# See https://github.com/containers/buildah/blob/843d15de3e797bd912607d27324d13a9d5c27dfb/imagebuildah/stage_executor.go#L61-L72 and
# for more details
RUN touch /test
