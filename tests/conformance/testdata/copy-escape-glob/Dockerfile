FROM scratch
WORKDIR /app
# The "[]" sequence immediately after the "[" signals to filepath.Glob() that
# it is _not_ the beginning of a "class" chunk in the globbing pattern.  N.B.
# this is a golang-specific way to do it, different from what glob(7) suggests.
COPY ./app/[[]xyz]/file.txt /app/
COPY ./app/[[]xyz]/[[]abc]/file.txt /app2/
COPY ./app/* /app3/
# This should only match one file.
COPY ./app/n\?pe/file.txt /app4/file.txt
# This should only match one file.
COPY ./app/st\*uv/file.txt /app5/file.txt
