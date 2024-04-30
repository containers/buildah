FROM busybox AS first
RUN mkdir /volumea /volumeb /volumec
RUN touch -r /bin/ls /volumea /volumeb /volumec
VOLUME /volumea
VOLUME /volumeb
VOLUME /volumec

FROM first
RUN chown 1000:1000 /volumea /volumeb /volumec
