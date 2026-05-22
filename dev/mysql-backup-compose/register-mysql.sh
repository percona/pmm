#!/bin/bash
set -euo pipefail

PMM_ADMIN=/usr/local/percona/pmm/bin/pmm-admin
PMM_ADMIN_USER="${PMM_ADMIN_USER:-admin}"
PMM_ADMIN_PASSWORD="${PMM_ADMIN_PASSWORD:-admin}"
PMM_SERVER_URL="https://${PMM_ADMIN_USER}:${PMM_ADMIN_PASSWORD}@127.0.0.1:8443/"
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-secret}"
STATE_DIR=/srv/mysql
LOG_PREFIX="[mysql-backup-register]"

log() {
	echo "${LOG_PREFIX} $*"
}

wait_for_pmm_server() {
	log "waiting for PMM Server..."
	until curl -skf https://127.0.0.1:8443/ping >/dev/null 2>&1; do
		sleep 2
	done
}

wait_for_pmm_agent() {
	log "waiting for pmm-agent..."
	local attempt=0
	until "${PMM_ADMIN}" status 2>/dev/null | grep -q 'Connected'; do
		attempt=$((attempt + 1))
		if [ "${attempt}" -ge 120 ]; then
			log "pmm-agent did not connect within 4 minutes"
			"${PMM_ADMIN}" status || true
			return 1
		fi
		sleep 2
	done
}

wait_for_mysqld() {
	log "waiting for mysqld..."
	local attempt=0
	until mysqladmin ping -h 127.0.0.1 -uroot --silent 2>/dev/null \
		|| mysqladmin ping -h 127.0.0.1 -uroot -p"${MYSQL_ROOT_PASSWORD}" --silent 2>/dev/null; do
		attempt=$((attempt + 1))
		if [ "${attempt}" -ge 120 ]; then
			log "mysqld did not become ready within 4 minutes"
			return 1
		fi
		sleep 2
	done
}

bootstrap_mysql_users() {
	if [ -f "${STATE_DIR}/.pmm-backup-initialized" ] && [ ! -f "${STATE_DIR}/.pmm-backup-root-password-set" ]; then
		log "setting MySQL root password"
		mysql -h 127.0.0.1 -uroot <<-EOSQL
			ALTER USER 'root'@'localhost' IDENTIFIED BY '${MYSQL_ROOT_PASSWORD}';
			FLUSH PRIVILEGES;
		EOSQL
		touch "${STATE_DIR}/.pmm-backup-root-password-set"
	fi

	if [ -f "${STATE_DIR}/.pmm-backup-root-password-set" ] && [ ! -f "${STATE_DIR}/.pmm-backup-user-created" ]; then
		log "creating pmm MySQL user"
		mysql -h 127.0.0.1 -uroot -p"${MYSQL_ROOT_PASSWORD}" < /opt/mysql-backup/mysql-init.sql
		touch "${STATE_DIR}/.pmm-backup-user-created"
	fi
}

register_mysql_service() {
	if "${PMM_ADMIN}" list --server-url="${PMM_SERVER_URL}" --server-insecure-tls 2>/dev/null | grep -q 'mysql-backup'; then
		log "mysql-backup service already registered"
		return 0
	fi

	log "registering mysql-backup service"
	local attempt=0
	while [ "${attempt}" -lt 30 ]; do
		if "${PMM_ADMIN}" add mysql mysql-backup 127.0.0.1:3306 \
			--server-url="${PMM_SERVER_URL}" \
			--server-insecure-tls \
			--username=pmm \
			--password=pmm \
			--query-source=perfschema \
			--cluster=backup-dev \
			--environment=dev; then
			log "mysql-backup registered successfully"
			return 0
		fi
		attempt=$((attempt + 1))
		log "pmm-admin add failed, retry ${attempt}/30"
		sleep 5
	done

	log "failed to register mysql-backup"
	"${PMM_ADMIN}" list --server-url="${PMM_SERVER_URL}" --server-insecure-tls || true
	return 1
}

wait_for_pmm_server
wait_for_pmm_agent
wait_for_mysqld
bootstrap_mysql_users
register_mysql_service
