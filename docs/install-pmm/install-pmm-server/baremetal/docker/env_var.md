# Environment variables

Use the following Docker container environment variables (with `-e var=value`) to set PMM Server parameters.

| Variable  &nbsp; &nbsp; &nbsp; &nbsp;                              | Description
| --------------------------------------------------------------- | -----------------------------------------------------------------------
| `DISABLE_UPDATES`                                               | Disables a periodic check for new PMM versions as well as ability to apply upgrades using the UI
| `DISABLE_TELEMETRY`                                             | Disable built-in telemetry and disable STT if telemetry is disabled.
| `METRICS_RESOLUTION`                                            | High metrics resolution in seconds.
| `METRICS_RESOLUTION_HR`                                         | High metrics resolution (same as above).
| `METRICS_RESOLUTION_MR`                                         | Medium metrics resolution in seconds.
| `METRICS_RESOLUTION_LR`                                         | Low metrics resolution in seconds.
| `DATA_RETENTION`                                                | The number of days to keep time-series data. <br />**N.B.** This must be set in a format supported by `time.ParseDuration` <br /> and represent the complete number of days. <br /> The supported units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, and `h`. <br /> The value must be a multiple of 24, e.g., for 90 days 2160h (90 * 24).
| `ENABLE_VM_CACHE`                                               | Enable cache in VM.
| `DISABLE_ALERTING`                           | Disables built-in Percona Alerting, which is enabled by default.
| `ENABLE_AZUREDISCOVER`                                          | Enable support for discovery of Azure databases.
| `DISABLE_BACKUP_MANAGEMENT`                                     | Disables Backup Management, which is enabled by default.
| `ENABLE_DBAAS`                                                  | Enable DBaaS features.
| `PMM_DEBUG`                                                     | Enables a more verbose log level.
| `PMM_TRACE`                                                     | Enables a more verbose log level including trace-back information.
| `PMM_PUBLIC_ADDRESS`                                            | External IP address or the DNS name on which PMM server is running.
| `PMM_WATCHTOWER_HOST=${PMM_WATCHTOWER_HOST:-http://watchtower:8080}` | Specifies the connection URL for the WatchTower container, including the schema (http), host (watchtower), and port (8080). 

The default value assumes that the WatchTower container is running on the same network as the PMM Server container and is accessible via the hostname watchtower.
| `PMM_WATCHTOWER_TOKEN=${PMM_WATCHTOWER_TOKEN:-123}`             | Defines the authentication token used for secure communication between the PMM Server container and the WatchTower container. Make sure this matches the value of the `WATCHTOWER_HTTP_API_TOKEN` environment variable set in the WatchTower container.

## Other variables

The following variables are also supported but values passed are not verified by PMM. If any other variable is found, it will be considered invalid and the server won't start.

| Variable                                                        | Description
| --------------------------------------------------------------- | ------------------------------------------------------
| `_`, `HOME`, `HOSTNAME`, `LANG`, `PATH`, `PWD`, `SHLVL`, `TERM` | Default environment variables.
| `GF_*`                                                          | [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/) environment variables.
| `VM_*`                                                          | [VictoriaMetrics'](https://docs.victoriametrics.com/Single-server-VictoriaMetrics.html#environment-variables) environment variables.
| `SUPERVISOR_`                                                   | `supervisord` environment variables.
| `KUBERNETES_`                                                   | Kubernetes environment variables.
| `MONITORING_`                                                   | Kubernetes monitoring environment variables.
| `PERCONA_TEST_`                                                 | Unknown variable but won't prevent the server starting.
| `PERCONA_TEST_DBAAS`                                            | Deprecated. Use `ENABLE_DBAAS`.