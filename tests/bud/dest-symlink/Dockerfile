FROM alpine

ENV HBASE_HOME="/usr/local/hbase"
ENV HBASE_CONF_DIR="/etc/hbase"

RUN mkdir $HBASE_HOME
RUN ln -s $HBASE_HOME $HBASE_CONF_DIR

COPY Dockerfile $HBASE_CONF_DIR
