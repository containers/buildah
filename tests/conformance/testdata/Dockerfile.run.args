FROM busybox
RUN echo first second
RUN /bin/echo third fourth
RUN ["/bin/echo", "fifth", "sixth"]
RUN ["/bin/sh", "-c", "echo inner $1", "", "outer"]