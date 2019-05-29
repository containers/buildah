FROM busybox
COPY . /upload/
COPY src /upload/src2/
RUN echo "CUT HERE"; /bin/find /upload | LANG=en_US.UTF-8 sort; echo "CUT HERE"
