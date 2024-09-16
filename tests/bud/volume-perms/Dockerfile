FROM alpine
VOLUME /vol/subvol
# At this point, the directory should exist, and it should have default
# permissions 0755, and we shouldn't get an error from trying to write to it
# because we it was created automatically.  If this image is built with the
# --compat-volumes flag, everything done after this point will be discarded.
RUN chmod 0711 /vol/subvol
RUN dd if=/dev/zero bs=512 count=1 of=/vol/subvol/subvolfile
