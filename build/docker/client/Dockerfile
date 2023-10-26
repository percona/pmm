FROM redhat/ubi9-minimal as users

RUN microdnf install shadow-utils jq -y

RUN groupadd -g 1002 pmm-agent && \
    useradd -u 1002 -r -g pmm-agent -s /sbin/nologin \
            -d /usr/local/percona/pmm3 \
            -c "PMM 2.X Client User" pmm-agent

FROM redhat/ubi9-micro

ARG VERSION
ARG BUILD_DATE

LABEL org.opencontainers.image.created ${BUILD_DATE}
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.title Percona Monitoring and Management Client
LABEL org.opencontainers.image.vendor Percona
LABEL org.opencontainers.image.version ${VERSION}

COPY LICENSE /licenses/
COPY --from=users --chown=0:0 /etc/passwd /etc/passwd
COPY --from=users --chown=0:0 /etc/group /etc/group

# Install jq along with its dependencies
COPY --from=users --chown=0:0 /usr/bin/jq /usr/bin/
COPY --from=users --chown=0:0 /usr/lib64/libjq.so.1.0.4 /lib64/libjq.so.1
COPY --from=users --chown=0:0 /usr/lib64/libonig.so.5.1.0 /lib64/libonig.so.5

RUN install -d -o pmm-agent -g pmm-agent -m 0775 /usr/local/percona/pmm3

ADD pmm3-client.tar.gz /tmp/

RUN cd /tmp/pmm3-client-* \
    && env PMM_USER=pmm-agent PMM_GROUP=root ./install_tarball \
    && cd /tmp \
    && rm -rf /tmp/pmm3-client*

USER pmm-agent
WORKDIR /usr/local/percona/pmm3/
ENV PATH /usr/local/percona/pmm3/bin/:$PATH

ENTRYPOINT ["/usr/local/percona/pmm3/bin/pmm-agent-entrypoint"]
