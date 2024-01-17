# The important thing about that first base image is that it has no history
# entries, but it does have at least one layer.

FROM nixery.dev/shell AS first-stage
RUN date > /date1.txt
RUN sleep 1 > /sleep1.txt

FROM first-stage
RUN date > /date2.txt
RUN sleep 1 > /sleep2.txt
