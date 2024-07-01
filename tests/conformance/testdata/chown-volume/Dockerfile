FROM busybox AS first
RUN mkdir /volumea /volumeb /volumec
# This next change will be omitted from naive diff output without
# both https://github.com/containers/storage/pull/1962 and
# https://github.com/containers/storage/pull/1968
RUN touch -r /bin/ls /volumea /volumeb /volumec
VOLUME /volumea
VOLUME /volumeb
VOLUME /volumec

FROM first
RUN chown 1000:1000 /volumea /volumeb /volumec
