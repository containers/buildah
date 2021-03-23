FROM centos:7
COPY --chown=1:2    script /usr/bin/script.12
COPY --chown=1:adm  script /usr/bin/script.1-adm
COPY --chown=1      script /usr/bin/script.1
COPY --chown=lp:adm script /usr/bin/script.lp-adm
COPY --chown=2:mail script /usr/bin/script.2-mail
COPY --chown=2      script /usr/bin/script.2
COPY --chown=bin    script /usr/bin/script.bin
COPY --chown=lp     script /usr/bin/script.lp
COPY --chown=3      script script2 /usr/local/bin/
RUN ls -al /usr/bin/script
