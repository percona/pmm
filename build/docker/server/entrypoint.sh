#!/bin/bash
# PMM Server Headless - Entrypoint
#
# Handles first-run initialization (PostgreSQL, SSL) and starts supervisord.
# Supports arbitrary UID via NSS wrapper for OpenShift/Kubernetes.

set -o errexit
set -o pipefail

PMM_DISTRIBUTION_METHOD="${PMM_DISTRIBUTION_METHOD:-docker}"
DIST_FILE="/srv/pmm-distribution"

log_info()  { echo "[INFO]  $(date -Iseconds) $*"; }
log_warn()  { echo "[WARN]  $(date -Iseconds) $*" >&2; }
log_error() { echo "[ERROR] $(date -Iseconds) $*" >&2; }

CURRENT_UID=$(id -u)
CURRENT_GID=$(id -g)
CURRENT_USER=$(whoami 2>/dev/null || echo "user-${CURRENT_UID}")

log_info "Starting PMM Server (headless)"
log_info "Running as UID:GID ${CURRENT_UID}:${CURRENT_GID}"

if [ ! -w /srv ]; then
    log_error "/srv is not writable for ${CURRENT_USER}"
    log_error "Fix: chown -R ${CURRENT_UID}:${CURRENT_GID} /srv"
    exit 1
fi

# NSS wrapper for arbitrary UID support (OpenShift)
if [ "$CURRENT_UID" != "1000" ] || [ "$CURRENT_GID" != "0" ]; then
    log_info "Setting up NSS wrapper for arbitrary UID support"

    NSS_WRAPPER_LIB="/usr/lib64/libnss_wrapper.so"
    if [ -f "$NSS_WRAPPER_LIB" ]; then
        NSS_WRAPPER_PASSWD=$(mktemp)
        NSS_WRAPPER_GROUP=$(mktemp)
        export NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP

        cleanup_nss() {
            rm -f "$NSS_WRAPPER_PASSWD" "$NSS_WRAPPER_GROUP" 2>/dev/null || true
        }
        trap cleanup_nss EXIT

        cat /etc/passwd > "$NSS_WRAPPER_PASSWD"
        cat /etc/group > "$NSS_WRAPPER_GROUP"

        if ! getent passwd "$CURRENT_UID" > /dev/null 2>&1; then
            echo "${CURRENT_USER}:x:${CURRENT_UID}:${CURRENT_GID}:PMM User:/srv:/bin/bash" >> "$NSS_WRAPPER_PASSWD"
        fi
        if ! getent group "$CURRENT_GID" > /dev/null 2>&1; then
            echo "pmmgroup:x:${CURRENT_GID}:" >> "$NSS_WRAPPER_GROUP"
        fi

        export LD_PRELOAD="${NSS_WRAPPER_LIB}${LD_PRELOAD:+:$LD_PRELOAD}"
        log_info "NSS wrapper enabled"
    else
        log_warn "NSS wrapper library not found, arbitrary UID may not work correctly"
    fi
fi

log_info "Ensuring directory structure"
mkdir -p /usr/share/pmm-server/nginx/{client_temp,proxy_temp,fastcgi_temp,uwsgi_temp,scgi_temp}
mkdir -p /srv/{logs,nginx,postgres14,clickhouse,victoriametrics,backup,prometheus/rules}
mkdir -p /srv/pmm-agent/tmp
mkdir -p /run/supervisor
chmod 770 /srv/pmm-agent/tmp 2>/dev/null || true

# First-run initialization
if [ ! -f "$DIST_FILE" ]; then
    log_info "First run detected - initializing /srv"
    echo -n "$PMM_DISTRIBUTION_METHOD" > "$DIST_FILE"

    log_info "Initializing PostgreSQL"
    /usr/pgsql-14/bin/initdb -D /srv/postgres14 --auth=trust --username=postgres

    log_info "Starting PostgreSQL to enable extensions"
    /usr/pgsql-14/bin/pg_ctl start -D /srv/postgres14 -w -t 30 -l /srv/logs/postgresql14.log

    /usr/pgsql-14/bin/psql -h /tmp -U postgres -c 'CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public'
    /usr/pgsql-14/bin/psql -h /tmp -U postgres -c "CREATE USER \"pmm-managed\" WITH PASSWORD 'pmm-managed'"
    /usr/pgsql-14/bin/psql -h /tmp -U postgres -c "CREATE DATABASE \"pmm-managed\" OWNER \"pmm-managed\""
    /usr/pgsql-14/bin/psql -h /tmp -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE \"pmm-managed\" TO \"pmm-managed\""

    /usr/pgsql-14/bin/pg_ctl stop -D /srv/postgres14 -w -t 30

    log_info "PostgreSQL initialized"
fi

chmod 750 /srv/postgres14 2>/dev/null || true

# SSL certificates
log_info "Generating SSL certificates"
mkdir -p /srv/nginx

if [ ! -f /srv/nginx/dhparam.pem ]; then
    cp /etc/nginx/ssl/dhparam.pem /srv/nginx/dhparam.pem
fi

if [ ! -f /srv/nginx/ca-certs.pem ]; then
    cp /etc/nginx/ssl/ca-certs.pem /srv/nginx/ca-certs.pem
fi

if [ ! -f /srv/nginx/certificate.conf ]; then
    cp /etc/nginx/ssl/certificate.conf /srv/nginx/certificate.conf
fi

if [ ! -f /srv/nginx/certificate.key ] || [ ! -f /srv/nginx/certificate.crt ]; then
    log_info "Creating self-signed SSL certificate"
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -extensions v3_req \
        -keyout /srv/nginx/certificate.key \
        -out /srv/nginx/certificate.crt \
        -config /srv/nginx/certificate.conf
fi

log_info "Validating nginx configuration"
if ! nginx -t 2>&1; then
    log_error "Nginx configuration test failed"
    exit 1
fi

log_info "Running pmm-managed-init"
if [ -x /usr/sbin/pmm-managed-init ]; then
    /usr/sbin/pmm-managed-init || log_warn "pmm-managed-init returned non-zero (may be expected)"
fi

log_info "Starting supervisord"
exec /usr/local/bin/supervisord -n -c /etc/supervisord.conf
