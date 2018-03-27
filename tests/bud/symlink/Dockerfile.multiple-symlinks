FROM alpine
RUN mkdir -p /data
RUN mkdir -p /test
RUN mkdir -p /test-log
RUN mkdir -p /myuser
RUN ln -s /test /myuser/log
RUN ln -s /test-log /test/bar
RUN ln -s /data/log /test-log/foo
VOLUME [ "/myuser/log/bar/foo/bin" ]
RUN echo "hello" > /data/log/blah.txt
