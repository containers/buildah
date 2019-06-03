FROM ubuntu as builder
RUN echo "symlink-test" > /bin/myexe.1 && ln -s /bin/myexe.1 /bin/myexe

FROM ubuntu
COPY --from=builder /bin/myexe /bin/
VOLUME [ "/bin" ]
