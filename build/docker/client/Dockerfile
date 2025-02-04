FROM redhat/ubi9-minimal AS users

RUN microdnf install shadow-utils jq tar -y

RUN groupadd -g 1002 pmm-agent && \
    useradd -u 1002 -r -g pmm-agent -s /sbin/nologin \
            -d /usr/local/percona/pmm \
            -c "PMM Client User" pmm-agent

FROM redhat/ubi9-micro

ARG VERSION
ARG RELEASE
ARG BUILD_DATE

LABEL build-date=${BUILD_DATE} \
      org.opencontainers.image.created=${BUILD_DATE} \
      org.opencontainers.image.licenses=Apache-2.0 \
      org.opencontainers.image.title="Percona Monitoring and Management Client" \
      org.opencontainers.image.vendor="Percona, LLC" \
      org.opencontainers.image.version=${VERSION} \
      version=${VERSION} \
      release=${RELEASE} \
      name="percona/pmm-client" \
      summary="PMM Client image" \
      description="The client image for Percona Monitoring and Management" \
      io.k8s.description="The client image for Percona Monitoring and Management" \
      io.k8s.display-name="Percona Monitoring and Management Client" \
      url="https://docs.percona.com/percona-monitoring-and-management" \
      vendor="Percona, LLC" \
      maintainer="Percona, LLC"

COPY LICENSE /licenses/
COPY --from=users --chown=0:0 /etc/passwd /etc/passwd
COPY --from=users --chown=0:0 /etc/group /etc/group

# Install jq along with its dependencies
COPY --from=users --chown=0:0 /usr/bin/jq /usr/bin/
COPY --from=users --chown=0:0 /usr/bin/tar /usr/bin/
COPY --from=users --chown=0:0 /usr/bin/curl /usr/bin/

COPY --from=users --chown=0:0 /usr/lib64/libjq.so.1.0.4 /lib64/libjq.so.1
COPY --from=users --chown=0:0 /usr/lib64/libonig.so.5.1.0 /lib64/libonig.so.5

# Dependencies of curl
COPY --from=users --chown=0:0 /usr/lib64/libkeyutils.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libz.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libkrb5support.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libkrb5.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libk5crypto.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libgssapi_krb5.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libcom_err.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libssl.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libcrypto.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libcurl.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libnghttp2.so* /lib64/
COPY --from=users --chown=0:0 /usr/lib64/libresolv.so* /lib64/

RUN install -d -o pmm-agent -g pmm-agent -m 0775 /usr/local/percona/pmm

ADD pmm-client.tar.gz /tmp/

RUN cd /tmp/pmm-client-* \
    && env PMM_USER=pmm-agent PMM_GROUP=root ./install_tarball \
    && cd /tmp \
    && rm -rf /tmp/pmm-client*

USER pmm-agent
WORKDIR /usr/local/percona/pmm/
ENV PATH=/usr/local/percona/pmm/bin/:$PATH

ENTRYPOINT ["/usr/local/percona/pmm/bin/pmm-agent-entrypoint"]
