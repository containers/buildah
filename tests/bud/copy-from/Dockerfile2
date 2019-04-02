FROM busybox AS basis
RUN echo hello > /newfile
FROM basis
RUN test -s /newfile
