FROM busybox
COPY . /upload/
COPY src /upload/src2/
COPY test1.txt /upload/test1.txt
RUN echo "CUT HERE"; /bin/find /upload | LANG=en_US.UTF-8 sort; echo "CUT HERE"
