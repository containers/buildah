FROM busybox
ADD https://github.com/openshift/origin/raw/master/README.md README.md
USER 1001
ADD https://github.com/openshift/origin/raw/master/LICENSE .
ADD https://github.com/openshift/origin/raw/master/LICENSE A
ADD https://github.com/openshift/origin/raw/master/LICENSE ./a
USER root
RUN mkdir ./b
ADD https://github.com/openshift/origin/raw/master/LICENSE ./b/a
ADD https://github.com/openshift/origin/raw/master/LICENSE ./b/.
ADD https://github.com/openshift/ruby-hello-world/archive/master.zip /tmp/
