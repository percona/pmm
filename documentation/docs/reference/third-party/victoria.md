# VictoriaMetrics

[VictoriaMetrics](https://victoriametrics.github.io/) is a third-party monitoring solution and time-series database.

## Push/Pull modes

VictoriaMetrics metrics data can be both 'pushed' to the server and 'pulled' by the server. When setting up services, you can decide which mode to use.

The mode (push/pull) is controlled by the `--metrics-mode` flag for the `pmm-admin config` and `pmm-admin add` commands.

If you need to change the metrics mode for an existing Service, you must remove it and re-add it with the same name and the required flags. (You cannot update a service.)

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

To set downsampling, use the `downsampling.period` parameter as follows:

```
-e VM_downsampling_period=20d:10m,120d:2h
```

This instructs VictoriaMetrics to [deduplicate](https://docs.victoriametrics.com/#deduplication) samples older than 20 days with 10 minute intervals and samples older than 120 days with two hour intervals.

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