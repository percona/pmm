ARG PMM_SERVER_IMAGE=perconalab/pmm-server-fb:PR-4368-377dfd9
FROM ${PMM_SERVER_IMAGE}

USER root

ARG MYSQL_SERVER_RPM_VERSION=8.4.6-6.1.el9
ARG XTRABACKUP_RPM_VERSION=8.4.0-4.1.el9

RUN dnf install -y epel-release https://repo.percona.com/yum/percona-release-latest.noarch.rpm \
	&& percona-release setup ps84lts \
	&& percona-release enable pxb-84-lts release \
	&& percona-release enable tools release \
	&& dnf install -y "percona-server-server-${MYSQL_SERVER_RPM_VERSION}" "percona-xtrabackup-84-${XTRABACKUP_RPM_VERSION}" qpress \
	&& dnf clean all \
	&& rm -rf /var/cache/dnf

COPY mysql-init.sql /opt/mysql-backup/mysql-init.sql
COPY start-mysqld.sh /usr/local/bin/start-mysqld.sh
COPY register-mysql.sh /usr/local/bin/register-mysql.sh
COPY supervisord-mysql.ini /etc/supervisord.d/mysql-backup.ini

RUN chmod +x /usr/local/bin/start-mysqld.sh /usr/local/bin/register-mysql.sh \
	&& install -d -m 0750 -o pmm -g root /srv/mysql-data /srv/mysql \
	&& rm -rf /var/lib/mysql \
	&& ln -s /srv/mysql-data /var/lib/mysql

USER 1000
