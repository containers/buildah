FROM alpine
RUN mkdir -p /vol/subvol/subsubvol
RUN dd if=/dev/zero bs=512 count=1 of=/vol/subvol/subsubvol/subsubvolfile
VOLUME /vol/subvol
# At this point, the contents below /vol/subvol may be frozen, so try to create
# something that will be discarded if it was.
RUN dd if=/dev/zero bs=512 count=1 of=/vol/subvol/subvolfile
# In particular, /vol/subvol/subvolfile should be wiped out if --compat-volumes
# behavior was selected.
RUN dd if=/dev/zero bs=512 count=1 of=/vol/volfile
# However, /vol/volfile should always exist, since /vol was not a volume, but
# we're making it one here.
VOLUME /vol
# And this should be redundant.
VOLUME /vol/subvol
# And now that we've frozen /vol, --compat-volumes should make this disappear,
# too.
RUN dd if=/dev/zero bs=512 count=1 of=/vol/anothervolfile

# ADD files which should persist, regardless of the --compat-volumes setting.
ADD Dockerfile /vol/Dockerfile
RUN stat /vol/Dockerfile
ADD Dockerfile /vol/Dockerfile2
RUN stat /vol/Dockerfile2

# This directory should still exist, since we cached /vol once it was declared
# a VOLUME, and /vol/subvol was created before that (as a VOLUME, but still).
RUN dd if=/dev/zero bs=512 count=1 of=/vol/subvol/subvolfile
