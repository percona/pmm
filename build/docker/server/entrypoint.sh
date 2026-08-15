#!/bin/bash
set -o errexit

declare PMM_DISTRIBUTION_METHOD="${PMM_DISTRIBUTION_METHOD:-docker}"
declare CURRENT_GID CURRENT_UID CURRENT_USER

# Returns 0 (true) if the given variable is set to "1" or "true".
is_enabled() { [ "$1" = "1" ] || [ "$1" = "true" ]; }
declare POSTGRES_DATA_DIR="/srv/postgres14"
declare POSTGRES_PASSWORD_FILE="/srv/.postgres_password"
declare POSTGRES_BIN_DIR="/usr/pgsql-14/bin"

# Get current user info - handle cases where user doesn't exist in passwd
CURRENT_UID=$(id -u)
CURRENT_GID=$(id -g)
if whoami &> /dev/null; then
    CURRENT_USER=$(whoami)
else
    CURRENT_USER="user-${CURRENT_UID}"
fi
echo "Running as UID ${CURRENT_UID}"

if [ ! -w /srv ]; then
    echo "FATAL: /srv is not writable for ${CURRENT_USER} user." >&2
    echo "Please make sure that /srv is owned by uid ${CURRENT_UID} and gid ${CURRENT_GID} and try again." >&2
    echo "You can change ownership by running: sudo chown -R ${CURRENT_UID}:${CURRENT_GID} /srv" >&2
    exit 1
fi

if [ "$CURRENT_UID" != "1000" ] || [ "$CURRENT_GID" != "0" ]; then
    echo "Running as UID:GID $CURRENT_UID:$CURRENT_GID, setting up for arbitrary UID..."

    # Try NSS wrapper first if available
    declare NSS_WRAPPER_LIB="/usr/lib64/libnss_wrapper.so"
    if [ ! -f "$NSS_WRAPPER_LIB" ]; then
        echo "Fatal: NSS wrapper library not found at $NSS_WRAPPER_LIB, exiting..."
        exit 1
    fi

    echo "Setting up NSS wrapper..."
    declare NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP
    # Set up NSS wrapper for arbitrary UID support
    NSS_WRAPPER_PASSWD=$(mktemp)
    NSS_WRAPPER_GROUP=$(mktemp)
    export NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP

    # Cleanup temp files on exit
    cleanup_nss_wrapper() {
        [ -f "$NSS_WRAPPER_PASSWD" ] && rm -f "$NSS_WRAPPER_PASSWD"
        [ -f "$NSS_WRAPPER_GROUP" ] && rm -f "$NSS_WRAPPER_GROUP"
    }
    trap cleanup_nss_wrapper EXIT

    # Copy existing passwd and group entries
    cat /etc/passwd > "$NSS_WRAPPER_PASSWD"
    cat /etc/group > "$NSS_WRAPPER_GROUP"

    # Add current user if not exists (suppress errors if NSS wrapper is not yet active)
    if ! getent passwd "$CURRENT_UID" > /dev/null 2>&1; then
        echo "${CURRENT_USER}:x:${CURRENT_UID}:${CURRENT_GID}:PMM User:/srv:/bin/bash" >> "$NSS_WRAPPER_PASSWD"
    fi

    # Add current group if not exists (suppress errors if NSS wrapper is not yet active)
    if ! getent group "$CURRENT_GID" > /dev/null 2>&1; then
        echo "${CURRENT_USER}:x:${CURRENT_GID}:" >> "$NSS_WRAPPER_GROUP"
    fi

    # Fix LD_PRELOAD assignment to avoid leading colon
    if [ -n "$LD_PRELOAD" ]; then
        export LD_PRELOAD="$NSS_WRAPPER_LIB:$LD_PRELOAD"
    else
        export LD_PRELOAD="$NSS_WRAPPER_LIB"
    fi
    echo "NSS wrapper enabled with $NSS_WRAPPER_LIB"
fi

# Initialize /srv if empty
declare DIST_FILE=/srv/pmm-distribution
if [ ! -f "$DIST_FILE" ]; then
    echo -n "$PMM_DISTRIBUTION_METHOD" > "$DIST_FILE"
    echo "Initializing /srv..."
    mkdir -p /srv/{backup,clickhouse,grafana/plugins,logs,nginx,prometheus/rules,victoriametrics}

    if is_enabled "$PMM_HA_ENABLE"; then
        echo "Skipping embedded PostgreSQL initialization in HA mode."
    elif is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES"; then
        echo "Skipping embedded PostgreSQL initialization (builtin PostgreSQL is disabled)."
    else
        echo "Initializing Postgres..."
        install -d -m 750 "$POSTGRES_DATA_DIR"

        # Generate a random password for postgres superuser
        declare POSTGRES_PASSWORD
        POSTGRES_PASSWORD=$(openssl rand -hex 16)

        # Store the password securely with restricted permissions
        echo -n "$POSTGRES_PASSWORD" > "$POSTGRES_PASSWORD_FILE"
        chmod 600 "$POSTGRES_PASSWORD_FILE"

        # Initialize database with password authentication
        "$POSTGRES_BIN_DIR/initdb" -D "$POSTGRES_DATA_DIR" --auth-host=scram-sha-256 --auth-local=trust --username=postgres --pwfile="$POSTGRES_PASSWORD_FILE"

        echo "Enabling pg_stat_statements extension for PostgreSQL..."
        "$POSTGRES_BIN_DIR/pg_ctl" start -D "$POSTGRES_DATA_DIR" -o "-c logging_collector=off"
        PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_BIN_DIR/psql" -U postgres -h /run/postgresql -d postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'
        "$POSTGRES_BIN_DIR/pg_ctl" stop -D "$POSTGRES_DATA_DIR"

        # Clean up password from environment
        unset POSTGRES_PASSWORD
    fi
fi

# Sync bundled Grafana plugins into /srv when the bundled set changes. 
# This must happen before supervisord starts.
declare PLUGINS_SRC=/usr/share/percona-dashboards/panels
declare PLUGINS_DST=/srv/grafana/plugins
declare PLUGINS_MARKER="$PLUGINS_DST/.pmm-synced-version"
declare BUNDLED_VERSION SYNCED_VERSION=""
BUNDLED_VERSION=$(< /usr/share/percona-dashboards/VERSION)
if [ -f "$PLUGINS_MARKER" ]; then
    SYNCED_VERSION=$(< "$PLUGINS_MARKER")
fi
if [ "$BUNDLED_VERSION" != "$SYNCED_VERSION" ]; then
    echo "Synchronizing Grafana plugins..."
    mkdir -p "$PLUGINS_DST"
    for panel in "$PLUGINS_SRC"/*/; do
        rm -rf "${PLUGINS_DST:?}/$(basename "$panel")"
    done
    cp -r "$PLUGINS_SRC"/* "$PLUGINS_DST"
    echo -n "$BUNDLED_VERSION" > "$PLUGINS_MARKER"
fi
unset PLUGINS_SRC PLUGINS_DST PLUGINS_MARKER BUNDLED_VERSION SYNCED_VERSION

echo "Creating nginx temp directories..."
mkdir -p /srv/nginx/tmp/{client,proxy,fastcgi,uwsgi,scgi}

if [ ! -d "/srv/pmm-agent/tmp" ]; then
    echo "Creating pmm-agent temp directory..."
    install -d -m 770 /srv/pmm-agent/tmp
fi

if is_enabled "$PMM_HA_ENABLE"; then
    echo "Skipping embedded PostgreSQL migration in HA mode."
elif is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES"; then
    echo "Skipping embedded PostgreSQL migration (builtin PostgreSQL is disabled)."
else
    mkdir -p /run/postgresql
    chmod 750 "$POSTGRES_DATA_DIR" || true
    # Scoped to this subshell so the helper scripts inherit them without polluting
    # the environment that supervisord and its children are started with.
    (
        export POSTGRES_DATA_DIR POSTGRES_PASSWORD_FILE POSTGRES_BIN_DIR
        bash /opt/ansible/roles/postgres/files/postgres-migration
        bash /opt/ansible/roles/postgres/files/postgres-sep
    )
fi

if is_enabled "$PMM_ENABLE_SEP" && { is_enabled "$PMM_HA_ENABLE" || is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES"; }; then
    echo "WARNING: ignoring PMM_ENABLE_SEP, the embedded PostgreSQL is not in use." >&2
fi

if is_enabled "$PMM_ENABLE_SEP" && { is_enabled "$PMM_HA_ENABLE" || is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES"; }; then
    echo "WARNING: not exposing a database to SEP, the embedded PostgreSQL is not in use." >&2
fi

# The reverse proxy is independent of which database SEP uses, so it is not nested
# in the embedded-PostgreSQL branch above.
declare SEP_NGINX_DIR=/etc/nginx/sep.d
declare SEP_NGINX_TEMPLATE=/opt/ansible/roles/nginx/files/sep/sep.conf.template
if is_enabled "$PMM_ENABLE_SEP"; then
    declare SEP_ADDRESS="${PMM_SEP_ADDRESS:-sep:9000}"
    # The address is interpolated into an nginx config, so an unvalidated value
    # is a config-injection vector. The digit count is capped so the range test
    # below cannot be handed a value that overflows the shell's integer parsing
    # and fails open.
    if ! [[ "$SEP_ADDRESS" =~ ^[A-Za-z0-9._-]+:[0-9]{1,5}$ ]]; then
        echo "FATAL: PMM_SEP_ADDRESS must be <host>:<port>, got '${SEP_ADDRESS}'." >&2
        exit 1
    fi
    # A variable proxy_pass resolves per request, so an out-of-range port would
    # pass nginx -t and only surface as a 502 at runtime.
    if [ "${SEP_ADDRESS##*:}" -lt 1 ] || [ "${SEP_ADDRESS##*:}" -gt 65535 ]; then
        echo "FATAL: PMM_SEP_ADDRESS port must be 1-65535, got '${SEP_ADDRESS##*:}'." >&2
        exit 1
    fi

    # Container DNS: 127.0.0.11 under Docker, an aardvark address under Podman.
    # IPv4 first, then an unscoped IPv6 in brackets -- nginx requires the brackets
    # and rejects a bare address, which fails nginx -t and so blocks the whole
    # server from starting.
    declare SEP_RESOLVER
    SEP_RESOLVER=$(awk '/^nameserver/ && $2 !~ /:/ { print $2; exit }' /etc/resolv.conf 2>/dev/null || true)
    if [ -z "$SEP_RESOLVER" ]; then
        # Scoped addresses are skipped rather than stripped of their zone: nginx has
        # no syntax for the interface scope, so a stripped fe80:: address yields a
        # config that passes nginx -t and can never route DNS -- trading a startup
        # failure for every /sep/ request timing out into the 503.
        SEP_RESOLVER=$(awk '/^nameserver/ && $2 !~ /%/ { print "[" $2 "]"; exit }' /etc/resolv.conf 2>/dev/null || true)
    fi
    if [ -z "$SEP_RESOLVER" ]; then
        if awk '/^nameserver/ && $2 ~ /%/ { found = 1 } END { exit !found }' /etc/resolv.conf 2>/dev/null; then
            echo "FATAL: /etc/resolv.conf lists only scoped IPv6 nameservers, such as fe80::1%eth0." >&2
            echo "nginx cannot express the interface scope, so such an address cannot be used." >&2
            echo "Please attach the container to a network with an IPv4 or unscoped IPv6 nameserver, or unset PMM_ENABLE_SEP." >&2
        else
            echo "FATAL: PMM_ENABLE_SEP is set but no nameserver found in /etc/resolv.conf." >&2
            echo "Please attach the container to a network with working DNS, or unset PMM_ENABLE_SEP." >&2
        fi
        exit 1
    fi

    if [ ! -f "$SEP_NGINX_TEMPLATE" ]; then
        echo "FATAL: missing ${SEP_NGINX_TEMPLATE}, cannot configure the SEP reverse proxy." >&2
        exit 1
    fi

    echo "Installing nginx reverse-proxy configuration for SEP at ${SEP_ADDRESS}..."
    mkdir -p "$SEP_NGINX_DIR"
    sed -e "s|__SEP_ADDRESS__|${SEP_ADDRESS}|" \
        -e "s|__SEP_RESOLVER__|${SEP_RESOLVER}|" \
        "$SEP_NGINX_TEMPLATE" > "$SEP_NGINX_DIR/sep.conf"
else
    # Clears the whole directory, not just the file this version writes: an older
    # build or an operator may have left others behind in the writable layer.
    rm -f "$SEP_NGINX_DIR"/*.conf
fi

# Not in grafana-sep: sep-provision is priority 20, so a probe firing before it starts would
# still see the previous run's marker. Fatal only when SEP is in use.
declare SEP_MARKER="/srv/.sep_provisioned"
rm -f "$SEP_MARKER" 2> /dev/null || true
if is_enabled "$PMM_ENABLE_SEP" && ! is_enabled "$PMM_HA_ENABLE" &&
    ! is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES" && [ -e "$SEP_MARKER" ]; then
    echo "FATAL: could not remove $SEP_MARKER; the health gate would report the previous provisioning run as this one." >&2
    exit 1
fi
unset SEP_MARKER

# Unconditional: the script owns its own gates, so the files it published are still
# removed on the start after PMM_ENABLE_SEP is cleared.
bash /opt/ansible/roles/sep/files/sep-secrets

# The last consumer has run, so drop the password before exec'ing supervisord: otherwise
# every child inherits it, and pmm-managed-init logs each variable it is handed - name and
# value - once PMM_TRACE is set.
unset PMM_SEP_POSTGRES_PASSWORD

echo "Generating self-signed certificates for nginx..."
bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate > /dev/null 2>&1

echo "Checking nginx configuration..."
if ! nginx -t -e /dev/stdout; then
    echo "Nginx configuration test failed, exiting..."
    exit 1
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

declare AGENT_CONFIG_DIR="/usr/local/percona/pmm/config"
declare AGENT_ID=pmm-server

if is_enabled "$PMM_HA_ENABLE"; then
    echo "High Availability mode is enabled."
    if [ -f "$AGENT_CONFIG_DIR/pmm-agent.yaml" ]; then
        rm -f "$AGENT_CONFIG_DIR/pmm-agent.yaml"
    fi

    AGENT_CONFIG_DIR="/srv/pmm-agent/config"
    if [ ! -d "$AGENT_CONFIG_DIR" ]; then
        echo "Creating pmm-agent config directory..."
        install -d -m 770 "$AGENT_CONFIG_DIR"
    fi

    AGENT_ID="$(uuidgen)"
fi

if [ ! -f "$AGENT_CONFIG_DIR/pmm-agent.yaml" ]; then
  echo "Creating pmm-agent configuration..."
  pmm-agent setup \
      --config-file="$AGENT_CONFIG_DIR/pmm-agent.yaml" \
      --skip-registration \
      --id="$AGENT_ID" \
      --paths-tempdir=/srv/pmm-agent/tmp \
      --paths-nomad-data-dir=/srv/nomad/data \
      --server-address=127.0.0.1:8443 \
      --server-insecure-tls
fi

unset AGENT_CONFIG_DIR AGENT_ID

# Start supervisor in foreground, i.e. as PID 1
exec supervisord -n -c /etc/supervisord.conf
