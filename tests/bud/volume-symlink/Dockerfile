FROM alpine
# Create symbolic links to simplify mounting
RUN mkdir -p /home/app/myvolume \
&& touch /home/app/myvolume/foo.txt \
&& ln -s /home/app/myvolume /config
VOLUME ["/config"]
