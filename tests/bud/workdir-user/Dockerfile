FROM alpine
RUN adduser -D http -h /home/http
USER http
WORKDIR /home/http/public
RUN stat -c '%u:%g %n' $PWD
RUN touch foobar
