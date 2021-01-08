# ProxySQL Server Exporter (proxysql_exporter)

The following options may be passed to the `proxysql:metrics` monitoring
service as additional options. For more information about this exporter see its
GitHub repository: [https://github.com/percona/proxysql_exporter](https://github.com/percona/proxysql_exporter).

## Collector Options

| Name                          | Description                                          |
| ----------------------------- | ---------------------------------------------------- |
| collect.mysql_connection_pool | Collect from stats_mysql_connection_pool.            |
| collect.mysql_status          | Collect from stats_mysql_global (SHOW MYSQL STATUS). |

## General Options

| Name               | Description |
| ------------------ | ----------------------------------------------------------------------- |
| log.format         | Set the log target and format. Example: “logger:syslog?appname=bob&local=7” or “logger:stdout?json=true” (default “logger:stderr”) |
| log.level          | Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal] |
| version            | Print version information and exit. |
| web.auth-file      | Path to YAML file with server_user, server_password options for http basic auth (overrides HTTP_AUTH env var). |
| web.listen-address | Address to listen on for web interface and telemetry. (default “:42004”) |
| web.ssl-cert-file  | Path to SSL certificate file. |
| web.ssl-key-file   | Path to SSL key file. |
| web.telemetry-path | Path under which to expose metrics. (default “/metrics”) |
