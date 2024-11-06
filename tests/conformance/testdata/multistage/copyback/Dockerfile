FROM mirror.gcr.io/alpine AS base
RUN touch -r /etc/os-release /1.txt

FROM mirror.gcr.io/alpine AS interloper
RUN --mount=type=bind,from=base,source=/,destination=/base,rw touch -r /etc/os-release /base/2.txt

FROM interloper
RUN --mount=type=bind,from=base,source=/,destination=/base mkdir /base2 && cp -a /base/*.txt /base2/ && touch -r /etc/os-release /base2 && ls -la /base2
