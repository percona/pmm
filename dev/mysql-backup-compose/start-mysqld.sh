#!/bin/bash
set -euo pipefail

DATADIR=/var/lib/mysql
SOCKET=/srv/mysql/mysql.sock
ERROR_LOG=/srv/logs/mysqld-backup-dev.err
PID_FILE=/srv/mysql/mysqld.pid
STATE_DIR=/srv/mysql
PORT=3306

install -d -m 0750 /srv/mysql-data "${STATE_DIR}"

if [ ! -d "${DATADIR}/mysql" ]; then
	rm -rf "${DATADIR:?}/"* "${DATADIR}"/.[!.]* "${DATADIR}"/..?* 2>/dev/null || true
	mysqld --initialize-insecure \
		--datadir="${DATADIR}" \
		--log-error="${ERROR_LOG}"
	touch "${STATE_DIR}/.pmm-backup-initialized"
fi

exec mysqld \
	--datadir="${DATADIR}" \
	--socket="${SOCKET}" \
	--log-error="${ERROR_LOG}" \
	--pid-file="${PID_FILE}" \
	--port="${PORT}" \
	--bind-address=127.0.0.1 \
	--server-id=1 \
	--log-bin=mysql-bin \
	--binlog-format=ROW \
	--performance-schema=ON
