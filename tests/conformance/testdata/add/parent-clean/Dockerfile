FROM mirror.gcr.io/busybox
RUN ln -s /symlink-target/subdirectory /symlink
ADD . /symlink/..
RUN find /symlink* -print
