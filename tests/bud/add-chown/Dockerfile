FROM alpine

ADD --chown=2367:3267 addchown.txt /tmp 
RUN stat -c "user:%u group:%g" /tmp/addchown.txt 
CMD /bin/sh

