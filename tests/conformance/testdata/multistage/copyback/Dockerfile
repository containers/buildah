FROM alpine AS base
RUN touch -r /etc/os-release /1.txt
FROM alpine AS interloper
RUN --mount=type=bind,from=base,source=/,destination=/base,rw touch -r /etc/os-release /base/2.txt
FROM base
RUN --mount=type=bind,from=interloper,source=/etc,destination=/etc2 touch -r /etc2/os-release /3.txt
