# PMM Server — Headless Build (No Grafana)
#
# Multi-stage build that compiles all Go binaries from the monorepo and
# downloads VictoriaMetrics, producing a self-contained server image.
#
# Usage:
#   docker build --platform linux/amd64 \
#     -f build/docker/server/Dockerfile \
#     -t pmm-headless:latest .

ARG GO_VERSION=1.25
ARG VM_VERSION=v1.93.4
ARG VERSION=3.5.0-do.1

# ============================================================================
# Stage 1: Build Go binaries
# ============================================================================
FROM golang:${GO_VERSION} AS builder

ARG VERSION

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w \
      -X github.com/percona/pmm/version.Version=${VERSION} \
      -X github.com/percona/pmm/version.PMMVersion=${VERSION} \
      -X github.com/percona/pmm/version.Timestamp=$(date -u +%s)" \
    -o /out/pmm-managed ./managed/cmd/pmm-managed/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w" -o /out/pmm-managed-init ./managed/cmd/pmm-managed-init/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w" -o /out/vmproxy ./vmproxy/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w" -o /out/percona-qan-api2 ./qan-api2/

ARG VM_VERSION
RUN curl -fsSL "https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/${VM_VERSION}/victoria-metrics-linux-amd64-${VM_VERSION}.tar.gz" \
    | tar xz -C /out/ && \
    mv /out/victoria-metrics-prod /out/victoriametrics

RUN curl -fsSL "https://github.com/VictoriaMetrics/VictoriaMetrics/releases/download/${VM_VERSION}/vmutils-linux-amd64-${VM_VERSION}.tar.gz" \
    | tar xz -C /out/ && \
    mv /out/vmalert-prod /out/vmalert && \
    rm -f /out/vmagent-prod /out/vmauth-prod /out/vmbackup-prod /out/vmrestore-prod /out/vmctl-prod

# ============================================================================
# Stage 2: Runtime image
# ============================================================================
FROM oraclelinux:9

ARG VERSION

ENV LANG=en_US.utf8 \
    LC_ALL=en_US.utf8 \
    PERCONA_TELEMETRY_DISABLE=1 \
    PS1="[pmm-server \\W] $ "

RUN dnf -y install epel-release && \
    dnf -y install https://repo.percona.com/yum/percona-release-latest.noarch.rpm && \
    percona-release enable ppg-14 release && \
    dnf -y install \
        percona-postgresql14-server \
        percona-postgresql14-contrib \
        percona-postgresql14 \
        python3-pip \
        python3-psycopg2 \
        nginx \
        openssl \
        curl \
        nss_wrapper \
        glibc-langpack-en \
        libcap && \
    pip3 install --no-cache-dir supervisor==4.2.4 && \
    dnf clean all && rm -rf /var/cache/dnf /var/cache/yum

RUN useradd -u 1000 -g 0 -d /srv -s /bin/bash pmm

# Go binaries from builder
COPY --from=builder /out/pmm-managed         /usr/sbin/pmm-managed
COPY --from=builder /out/pmm-managed-init    /usr/sbin/pmm-managed-init
COPY --from=builder /out/vmproxy             /usr/sbin/vmproxy
COPY --from=builder /out/percona-qan-api2    /usr/sbin/percona-qan-api2
COPY --from=builder /out/victoriametrics     /usr/sbin/victoriametrics
COPY --from=builder /out/vmalert             /usr/sbin/vmalert

# Nginx configuration
COPY build/ansible/roles/nginx/files/nginx.conf           /etc/nginx/nginx.conf
COPY build/ansible/roles/nginx/files/conf.d/pmm.conf      /etc/nginx/conf.d/pmm.conf
COPY build/ansible/roles/nginx/files/conf.d/pmm-ssl.conf  /etc/nginx/conf.d/pmm-ssl.conf
COPY build/ansible/roles/nginx/files/ssl/ca-certs.pem     /etc/nginx/ssl/ca-certs.pem
COPY build/ansible/roles/nginx/files/ssl/certificate.conf /etc/nginx/ssl/certificate.conf
COPY build/ansible/roles/nginx/files/ssl/dhparam.pem      /etc/nginx/ssl/dhparam.pem
COPY build/ansible/roles/nginx/files/ssl/generate-ssl-certificate \
     /var/lib/cloud/scripts/per-boot/generate-ssl-certificate

# Supervisord configuration
RUN mkdir -p /etc/supervisord.d /run/supervisor
COPY build/docker/server/configs/supervisord.conf /etc/supervisord.conf
COPY build/docker/server/configs/pmm.ini          /etc/supervisord.d/pmm.ini
RUN chown -R pmm:0 /etc/supervisord.d /run/supervisor

# Directory structure
RUN mkdir -p \
        /srv/prometheus/rules \
        /srv/logs \
        /srv/nginx \
        /srv/victoriametrics/data \
        /srv/backup \
        /srv/pmm-agent/tmp \
        /run/postgresql \
        /usr/share/pmm-server/static \
        /usr/share/pmm-server/nginx/client_temp \
        /usr/share/pmm-server/nginx/proxy_temp \
        /usr/share/pmm-server/nginx/fastcgi_temp \
        /usr/share/pmm-server/nginx/uwsgi_temp \
        /usr/share/pmm-server/nginx/scgi_temp \
        /usr/local/percona/pmm/config \
        /usr/local/percona/pmm/tools \
        /var/lib/cloud/scripts/per-boot && \
    touch /etc/victoriametrics-promscrape.yml \
          /srv/logs/nginx.log \
          /srv/logs/postgresql14.log \
          /srv/logs/supervisord.log \
          /run/nginx.pid && \
    chown -R pmm:0 \
        /srv \
        /run/postgresql \
        /run/nginx.pid \
        /usr/share/pmm-server \
        /etc/nginx \
        /etc/supervisord.d \
        /etc/victoriametrics-promscrape.yml \
        /usr/local/percona/pmm && \
    chmod -R g+rwX /srv /run/postgresql /run/supervisor

COPY build/docker/server/entrypoint.sh /opt/entrypoint.sh
RUN chmod +x /opt/entrypoint.sh

RUN echo "${VERSION}" > /srv/pmm-version

LABEL org.opencontainers.image.title="PMM Server Headless" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.vendor="DigitalOcean" \
      org.opencontainers.image.description="PMM Server built from source without Grafana for DBaaS metrics collection"

USER 1000
VOLUME ["/srv"]
EXPOSE 8080/tcp 8443/tcp

HEALTHCHECK --interval=3s --timeout=2s --start-period=30s --retries=3 \
    CMD curl -sf http://127.0.0.1:8080/v1/server/readyz || exit 1

CMD ["/opt/entrypoint.sh"]
