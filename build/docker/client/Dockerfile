FROM redhat/ubi9-minimal as users

RUN microdnf install shadow-utils

RUN groupadd -g 1002 pmm-agent && \
    useradd -u 1002 -r -g pmm-agent -s /sbin/nologin \
            -d /usr/local/percona/pmm2 \
            -c "PMM 2.X Client User" pmm-agent

FROM redhat/ubi9-micro

ARG VERSION
ARG BUILD_DATE

LABEL org.label-schema.build-date ${BUILD_DATE}
LABEL org.label-schema.license Apache-2.0
LABEL org.label-schema.name Percona Monitoring and Management Client
LABEL org.label-schema.vendor Percona
LABEL org.label-schema.version ${VERSION}
LABEL org.label-schema.schema-version "1.0"
LABEL org.opencontainers.image.created ${BUILD_DATE}
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.title Percona Monitoring and Management Client
LABEL org.opencontainers.image.vendor Percona
LABEL org.opencontainers.image.version ${VERSION}

COPY LICENSE /licenses/
COPY --from=users --chown=0:0 /etc/passwd /etc/passwd
COPY --from=users --chown=0:0 /etc/group /etc/group

RUN install -d -o pmm-agent -g pmm-agent -m 0775 /usr/local/percona/pmm2

ADD pmm2-client.tar.gz /tmp/

RUN cd /tmp/pmm2-client-* \
    && env PMM_USER=pmm-agent PMM_GROUP=root ./install_tarball \
    && cd /tmp \
    && rm -rf /tmp/pmm2-client*

USER pmm-agent
WORKDIR /usr/local/percona/pmm2/
ENV PATH /usr/local/percona/pmm2/bin/:$PATH

ENTRYPOINT ["/usr/local/percona/pmm2/bin/pmm-agent-entrypoint"]
