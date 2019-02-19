FROM alpine
ARG USED_ARG="used_value"
RUN echo ${USED_ARG}
FROM scratch
COPY --from=0 /etc/passwd /root/passwd-file
