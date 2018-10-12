FROM alpine

COPY --chown=2367:3267 copychown.txt /tmp 
RUN stat -c "user:%u group:%g" /tmp/copychown.txt 
CMD /bin/sh

