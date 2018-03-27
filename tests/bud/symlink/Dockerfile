FROM alpine
RUN mkdir -p /data
RUN ln -s /test-log /blah
RUN ln -s /data/log /test-log
VOLUME [ "/test-log/test" ]
RUN echo "hello" > /data/log/blah.txt
