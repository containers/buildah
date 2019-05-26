FROM ubuntu as builder
FROM ubuntu
COPY --from=builder /bin/bash my/bin/bash
RUN stat -c "permissions=%a" my/bin
