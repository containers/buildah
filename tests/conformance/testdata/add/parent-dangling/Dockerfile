FROM mirror.gcr.io/busybox
RUN ln -s symlink-target /symlink
ADD . /symlink/subdirectory/
RUN find /symlink* -print
