FROM alpine
RUN --mount=type=secret,id=MYSECRET \
     printf "%s\n" $(cat /run/secrets/MYSECRET)
