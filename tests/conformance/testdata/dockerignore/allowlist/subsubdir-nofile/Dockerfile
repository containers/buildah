FROM busybox
RUN touch /file
FROM scratch
COPY --from=0 /file /
ADD . .
