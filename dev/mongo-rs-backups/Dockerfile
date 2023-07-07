FROM percona/percona-server-mongodb:4.2

USER root

RUN yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm && \
    percona-release enable pbm release && \
    percona-release enable pmm2-client release

RUN yum install -y percona-backup-mongodb pmm2-client

