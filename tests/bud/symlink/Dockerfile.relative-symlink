FROM alpine
RUN mkdir -p /data
RUN ln -s ../log /test-log
VOLUME [ "/test-log/test" ]
RUN ln -s ../data /var/data
RUN touch /data/empty
VOLUME [ "/var/data" ]
RUN pwd
