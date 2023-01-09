FROM alpine
RUN mkdir test
WORKDIR test
RUN --mount=type=secret,id=secret-foo,dst=secret1.txt --mount=type=secret,id=secret-bar,dst=secret2.txt \
     cat /test/secret1.txt && cat /test/secret2.txt
