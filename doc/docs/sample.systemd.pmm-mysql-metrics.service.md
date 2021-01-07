# Examples of the **systemd** Unit File

This page contains examples of setting up the **systemd** unit file.

## Default systemd unit file with SSL related options highlighted

If the **systemd** unit file contains options related to SSL the
communication between the Prometheus exporter and the monitored
system occurs via the HTTPS protocol.

```
[Unit]
Description=PMM Prometheus mysqld_exporter 42002
ConditionFileIsExecutable=/usr/local/percona/pmm-client/mysqld_exporter
After=network.target
After=syslog.target

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart=/bin/sh -c '/usr/local/percona/pmm-client/mysqld_exporter \
-collect.auto_increment.columns=true \
-collect.binlog_size=true \
-collect.global_status=true \
-collect.global_variables=true \
-collect.info_schema.innodb_metrics=true \
-collect.info_schema.processlist=true \
-collect.info_schema.query_response_time=true \
-collect.info_schema.tables=true \
-collect.info_schema.tablestats=true \
-collect.info_schema.userstats=true \
-collect.perf_schema.eventswaits=true \
-collect.perf_schema.file_events=true \
-collect.perf_schema.indexiowaits=true \
-collect.perf_schema.tableiowaits=true \
-collect.perf_schema.tablelocks=true \
-collect.slave_status=true \
-web.listen-address=172.17.0.1:42002 \
-web.auth-file=/usr/local/percona/pmm-client/pmm.yml \
-web.ssl-cert-file=/usr/local/percona/pmm-client/server.crt \
-web.ssl-key-file=/usr/local/percona/pmm-client/server.key >> /var/log/pmm-mysql-metrics-42002.log 2>&1'

Environment="DATA_SOURCE_NAME=pmm:a7NB_9e14SO;,s-O5e,q@unix(/var/run/mysqld/mysqld.sock)/?parseTime=true&time_zone='%2b00%3a00'&loc=UTC"

Restart=always
RestartSec=120

[Install]
WantedBy=multi-user.target
```

Remove the SSL related options to disable HTTPS for the exporter.

```
[Unit]
Description=PMM Prometheus mysqld_exporter 42002
ConditionFileIsExecutable=/usr/local/percona/pmm-client/mysqld_exporter
After=network.target
After=syslog.target

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart=/bin/sh -c '/usr/local/percona/pmm-client/mysqld_exporter \
-collect.auto_increment.columns=true \
-collect.binlog_size=true \
-collect.global_status=true \
-collect.global_variables=true \
-collect.info_schema.innodb_metrics=true \
-collect.info_schema.processlist=true \
-collect.info_schema.query_response_time=true \
-collect.info_schema.tables=true \
-collect.info_schema.tablestats=true \
-collect.info_schema.userstats=true \
-collect.perf_schema.eventswaits=true \
-collect.perf_schema.file_events=true \
-collect.perf_schema.indexiowaits=true \
-collect.perf_schema.tableiowaits=true \
-collect.perf_schema.tablelocks=true \
-collect.slave_status=true \
-web.listen-address=172.17.0.1:42002 \
-web.auth-file=/usr/local/percona/pmm-client/pmm.yml \

>> /var/log/pmm-mysql-metrics-42002.log 2>&1'

Environment="DATA_SOURCE_NAME=pmm:a7NB_9e14SO;,s-O5e,q@unix(/var/run/mysqld/mysqld.sock)/?parseTime=true&time_zone='%2b00%3a00'&loc=UTC"

Restart=always
RestartSec=120

[Install]
WantedBy=multi-user.target
```

<!-- -*- mode: rst -*- -->
<!-- Tips (tip) -->
<!-- Abbreviations (abbr) -->
<!-- Docker commands (docker) -->
<!-- Graphical interface elements (gui) -->
<!-- Options and parameters (opt) -->
<!-- pmm-admin commands (pmm-admin) -->
<!-- SQL commands (sql) -->
<!-- PMM Dashboards (dbd) -->
<!-- * Text labels -->
<!-- Special headings (h) -->
<!-- Status labels (status) -->
