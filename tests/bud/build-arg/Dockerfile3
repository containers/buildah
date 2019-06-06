FROM busybox
MAINTAINER jdoe <jdoe@example.com>
ENV container="docker"

RUN echo this-should-be-cached-but-it-s-not

ARG USERNAME
ARG UID
ARG CODE
ARG PGDATA
ARG PORT=55555

CMD ["/container-run"]
