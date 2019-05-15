FROM busybox:latest AS builder
ENV "BUILD_LOGLEVEL"="5"
RUN touch /tmp/preCommit
ENTRYPOINT /bin/sleep 600
ENV "OPENSHIFT_BUILD_NAME"="mydockertest-1" "OPENSHIFT_BUILD_NAMESPACE"="default"
LABEL "io.openshift.build.name"="mydockertest-1" "io.openshift.build.namespace"="default"

FROM builder
ENV "BUILD_LOGLEVEL"="5"
RUN touch /tmp/postCommit

FROM builder
ENV "BUILD_LOGLEVEL"="5"
