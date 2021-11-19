FROM alpine
# Will stall if this is connected to a terminal, or fail if it's not readable
RUN cat        /dev/stdin
# Will fail if it's not writable
RUN echo foo > /dev/stdout
# Will fail if it's not writable
RUN echo foo > /dev/stderr
