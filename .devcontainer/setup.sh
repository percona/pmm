#!/bin/bash

# See CONTRIBUTING.md.

set -o errexit
set -o nounset
set -o pipefail

declare MARKER="/tmp/devcontainer-setup-done"
declare POSTGRES_DATA_DIR="/srv/postgres14"
declare HBA_RULE="host    all         all     all     trust"

# Installs required and useful RPM packages.
install_packages() {
    dnf install -y --enablerepo=ol9_codeready_builder \
        gcc git make pkgconfig \
        vim \
        psmisc procps lsof diffutils \
        man man-pages \
        ansible-lint glibc-static \
        openssl-devel \
        krb5-devel
}

# Installs Go toolchain.
install_go() {
    if [ -z "${GO_VERSION:-}" ]; then
        echo "GO_VERSION is not set" >&2
        exit 1
    fi

    curl -sS https://raw.githubusercontent.com/travis-ci/gimme/v1.5.6/gimme -o /usr/local/bin/gimme
    chmod +x /usr/local/bin/gimme

    local go_version
    go_version=$(gimme -r "$GO_VERSION")

    gimme "$go_version"
    rm -fr /usr/local/go
    mv -f "/root/.gimme/versions/go${go_version}.linux.amd64" /usr/local/go
    update-alternatives --install '/usr/bin/go' 'go' '/usr/local/go/bin/go' 0
    update-alternatives --set go /usr/local/go/bin/go
    update-alternatives --install '/usr/bin/gofmt' 'gofmt' '/usr/local/go/bin/gofmt' 0
    update-alternatives --set gofmt /usr/local/go/bin/gofmt
    mkdir -p /root/go/bin
    go version
    go env
}

# Installs Node.js 22 and Yarn.
install_node() {
    dnf module enable -y nodejs:22
    dnf install -y nodejs npm
    npm install -g yarn@1.22.22
    node --version
    yarn --version
}

# Tweaks the embedded PostgreSQL configuration for development. Idempotent.
configure_postgres() {
    local conf="$POSTGRES_DATA_DIR/postgresql.conf"
    local hba="$POSTGRES_DATA_DIR/pg_hba.conf"
    local changed=0
    local original

    # /srv is empty in the server image: PostgreSQL is initialized by the entrypoint on the first
    # container start. Hence, this is a no-op during the image build.
    if [ ! -f "$conf" ]; then
        echo "$conf does not exist, skipping PostgreSQL configuration."
        return
    fi

    echo "Configuring $conf..."
    original=$(< "$conf")
    # Allow connecting from any host, needed to connect from host to PG running in docker.
    # Also turn fsync off: create database operations with fsync on are very slow on Ubuntu,
    # and having fsync off in a dev environment is fine.
    sed -i \
        -e "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" \
        -e "s/#fsync = on/fsync = off/" \
        "$conf"
    if [ "$original" != "$(< "$conf")" ]; then
        changed=1
    fi

    # Allow password-less connections from any host (dev environment only). pg_hba.conf is
    # first-match-wins, so the rule goes above the ones initdb generates for loopback.
    if ! grep -qF "$HBA_RULE" "$hba"; then
        echo "Adding a rule to $hba..."
        sed -i -e "0,/^host/s|^host|$HBA_RULE\nhost|" "$hba"
        # fall back to appending when the file has no host rules to insert before
        grep -qF "$HBA_RULE" "$hba" || echo "$HBA_RULE" >> "$hba"
        changed=1
    fi

    if [ "$changed" -eq 1 ]; then
        echo "Restarting PostgreSQL..."
        # A failed restart is not fatal: the new configuration is picked up on the next start.
        supervisorctl restart postgresql ||
            echo "Failed to restart PostgreSQL, restart the container to apply the changes."
    fi
}

declare start=$SECONDS

# PostgreSQL is initialized on the first container start, hence its configuration is applied
# on every run, unlike the toolchain installation below.
if [ "${1:-}" != "--postgres-only" ]; then
    if [ -f "$MARKER" ]; then
        echo "$MARKER exists, skipping the installation."
    else
        install_packages
        install_go
        install_node
        make init
        touch "$MARKER"
    fi
fi

configure_postgres
echo "Done in $((SECONDS - start))s"
