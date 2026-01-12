FROM alpine
RUN mkdir /hello
VOLUME /var/lib/testdata
RUN touch file.txt
EXPOSE 8080
ARG HTTP_SERVER_PORT
ADD http://0.0.0.0:${HTTP_SERVER_PORT}/README.md /tmp/
ENV foo=bar
