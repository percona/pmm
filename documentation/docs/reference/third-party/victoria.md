# VictoriaMetrics

[VictoriaMetrics](https://victoriametrics.github.io/) is a third-party monitoring solution and time-series database.

## Push/Pull modes

VictoriaMetrics metrics data can be both 'pushed' to the server and 'pulled' by the server. When setting up services, you can decide which mode to use.

The mode (push/pull) is controlled by the `--metrics-mode` flag for the `pmm-admin config` and `pmm-admin add` commands.

To change the metrics mode of an existing exporter, run `pmm-admin inventory change agent <AGENT_TYPE> <AGENT_ID> --push-metrics` for push mode, or `--push-metrics=false` for pull mode. See [`pmm-admin inventory change agent`](../../use/commands/pmm-admin/inventory.md#pmm-admin-inventory-change-agent).

## Remapped targets for direct Prometheus paths

Direct Prometheus paths return structured information directly from Prometheus, bypassing the PMM application.

They are accessed by requesting a URL of the form `<PMM SERVER URL>/prometheus/<PATH>`.

As a result of the move to VictoriaMetrics some direct Prometheus paths are no longer available.

| Prometheus path                 | VictoriaMetrics equivalent
|---------------------------------|--------------------------------------------------------------------------------------------------------------------------
| `/prometheus/alerts`            | No change.
| `/prometheus/config`            | No equivalent, but there is some information at `/prometheus/targets`.
| `/prometheus/flags`             | The `flag` metrics at `/prometheus/metrics`.
| `/prometheus/graph`             | `/graph/explore` (Grafana) or `graph/d/prometheus-advanced/advanced-data-exploration` (PMM dashboard).
| `/prometheus/rules`             | No change.
| `/prometheus/service-discovery` | No equivalent.
| `/prometheus/status`            | Some information at `/prometheus/metrics`. High cardinality metrics information at `/prometheus/api/v1/status/tsdb`.
| `/prometheus/targets`           | `/victoriametrics/targets`

## Environment variables

PMM predefines certain flags that allow users to set all other [VictoriaMetrics parameters](https://docs.victoriametrics.com/#list-of-command-line-flags) as environment variables:

The environment variable must be prepended with `VM_`.

### Example

To let queries run for up to two minutes, use the `search.maxQueryDuration` parameter as follows:

```
-e VM_search_maxQueryDuration=120s
```

This instructs VictoriaMetrics to stop a query that runs for longer than 120 seconds. The default in PMM is 90 seconds.

## Downsampling and Prometheus data files

Starting with PMM 3.12.0, PMM Server includes the official open source release of VictoriaMetrics. This release doesn't include downsampling and doesn't read Prometheus data files.

If one of the following environment variables is set, PMM ignores it and logs a warning at startup:

- `VM_downsampling_period`: new metrics stay at full resolution, so they need more disk space. Data that is already downsampled stays downsampled.
- `VM_prometheusDataPath`: metrics stored in Prometheus data files no longer appear in PMM.

### Limit disk use without downsampling

Downsampling saves disk space by keeping fewer samples of older data. Without downsampling, PMM stores more samples. For example, a PMM Server that used `VM_downsampling_period=20d:10m` with the default 30-day retention stores about 1.5 times as many samples. A PMM Server that used `VM_downsampling_period=20d:10m,120d:2h` with one year of retention stores more than 10 times as many samples.

To limit disk use, choose one of the following options:

- Set `VM_dedup_minScrapeInterval` to keep one sample per interval for each time series. This setting applies to all data, including the most recent metrics.
- Collect metrics less often with the [metrics resolution](../../configure-pmm/metrics_res.md) settings.
- Shorten the [data retention](../../configure-pmm/advanced_settings.md#data-retention) period, or add disk space.
- Send metrics to an [external VictoriaMetrics Enterprise database](#using-victoriametrics-external-database-instance), which is a Technical Preview feature.

### Import Prometheus data files

To import metrics that are stored in Prometheus data files, use the [`vmctl prometheus`](https://docs.victoriametrics.com/victoriametrics/vmctl/prometheus/) command, which reads a Prometheus snapshot and writes its metrics to VictoriaMetrics. PMM Server doesn't include `vmctl`. To get `vmctl`, download the `vmutils` archive from the [VictoriaMetrics releases](https://github.com/VictoriaMetrics/VictoriaMetrics/releases) page.

## Using VictoriaMetrics external database instance

!!! caution alert alert-warning "Important/Caution"
    This feature is still in [Technical Preview](../../reference/glossary.md#technical-preview) and is subject to change. We recommend that early adopters use this feature for evaluation purposes only.

You can use an external VictoriaMetrics database for monitoring in PMM.

The environment variable `PMM_VM_URL` has been added, which should point to the external VictoriaMetrics database and should have the following format:

```sh
http(s)://hostname:port/path.
```

If the external VictoriaMetrics database requires basic authentication, the following environment variables should be used. 

```sh
VMAGENT_remoteWrite_basicAuth_username={username}
VMAGENT_remoteWrite_basicAuth_password={password}
```

These credentials can be [set on PMM Server](../../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#configure-vmagent-variables) and will automatically apply to all connected PMM Clients.

If other authentication methods are used on the VictoriaMetrics side, use any of the `vmagent` environment variables by prepending `VMAGENT_ prefix`.

When external VictoriaMetrics is configured, internal VictoriaMetrics stops. In this case, VM Agent on PMM Server pulls metrics from agents configured in the `pull metrics mode` and from remote nodes. Data is then pushed to external VictoriaMetrics.

!!! note alert alert-primary "Note"
    VM Agents run by PMM Clients push data directly to external VictoriaMetrics. 
    
    Ensure that they can connect to external VictoriaMetrics.

## Troubleshooting

To troubleshoot issues, see the VictoriaMetrics [troubleshooting documentation](https://victoriametrics.github.io/#troubleshooting).

You can also contact the VictoriaMetrics team via:

- [Google Groups](https://groups.google.com/forum/#!forum/victorametrics-users)
- [Slack](http://slack.victoriametrics.com/)
- [Reddit](https://www.reddit.com/r/VictoriaMetrics/)
- [Telegram](https://t.me/VictoriaMetrics_en)