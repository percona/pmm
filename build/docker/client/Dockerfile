FROM redhat/ubi9-minimal AS builder

ENV PS_VERSION=8.0.40-31.1
ENV ARCH=x86_64

RUN microdnf install shadow-utils wget jq tar -y

RUN wget https://repo.percona.com/ps-80/yum/release/9/RPMS/x86_64/percona-server-shared-$PS_VERSION.el9.$ARCH.rpm https://repo.percona.com/ps-80/yum/release/9/RPMS/x86_64/percona-server-client-$PS_VERSION.el9.$ARCH.rpm

RUN rpm -i percona-server-shared-$PS_VERSION.el9.$ARCH.rpm percona-server-client-$PS_VERSION.el9.$ARCH.rpm

# Install required dependencies into the installation root
RUN microdnf install -y \
        perl-DBD-MySQL \
        perl-DBI \
        perl-Digest-MD5 \
        perl-English \
        perl-FindBin \
        perl-IO-Socket-SSL \
        perl-Sys-Hostname \
        perl-TermReadKey \
        perl-Time-HiRes \
        perl-sigtrap \
    && rm -rf /var/cache/*

RUN ls -R /usr/bin

RUN groupadd -g 1002 pmm-agent && \
    useradd -u 1002 -r -g pmm-agent -s /sbin/nologin \
            -d /usr/local/percona/pmm \
            -c "PMM Client User" pmm-agent

# ADD pmm-client.tar.gz /tmp/

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
COPY --from=builder --chown=0:0 /etc/passwd /etc/passwd
COPY --from=builder --chown=0:0 /etc/group /etc/group

# Install jq along with its dependencies
COPY --from=builder --chown=0:0 /usr/bin/jq /usr/bin/
COPY --from=builder --chown=0:0 /usr/bin/tar /usr/bin/
COPY --from=builder --chown=0:0 /usr/bin/curl /usr/bin/

COPY --from=builder --chown=0:0 /usr/lib64/libjq.so.1.0.4 /lib64/libjq.so.1
COPY --from=builder --chown=0:0 /usr/lib64/libonig.so.5.1.0 /lib64/libonig.so.5

# Install perl along with its dependencies
COPY --from=builder --chown=0:0 /usr/bin/mysql /usr/bin/
COPY --from=builder --chown=0:0 /usr/bin/perl /usr/bin/

COPY --from=builder --chown=0:0 /usr/lib64/*perl* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libcrypt.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libstdc++.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libreadline.so* /usr/lib64/
COPY --from=builder --chown=0:0 /etc/ld.so.conf.d/mysql-x86_64.conf /etc/ld.so.conf.d

# Dependencies of curl
COPY --from=builder --chown=0:0 /usr/lib64/libkeyutils.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libz.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libkrb5support.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libkrb5.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libk5crypto.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libgssapi_krb5.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libcom_err.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libssl.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libcrypto.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libcurl.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libnghttp2.so* /usr/lib64/
COPY --from=builder --chown=0:0 /usr/lib64/libresolv.so* /usr/lib64/

# COPY --from=builder --chown=0:0 /tmp/pmm-client* /tmp/

RUN install -d -o pmm-agent -g pmm-agent -m 0775 /usr/local/percona/pmm

ADD pmm-client.tar.gz /tmp/
RUN cd /tmp/pmm-client* \
    && env PMM_USER=pmm-agent PMM_GROUP=root ./install_tarball \
    && cd /tmp \
    && rm -rf /tmp/pmm-client*

USER pmm-agent
WORKDIR /usr/local/percona/pmm/
ENV PATH=/usr/local/percona/pmm/bin/:$PATH

ENTRYPOINT ["/usr/local/percona/pmm/bin/pmm-agent-entrypoint"]
