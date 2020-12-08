# VictoriaMetrics

[VictoriaMetrics](https://victoriametrics.github.io/) is a third-party monitoring solution and time-series database that replaced Prometheus in [PMM 2.12.0](../release-notes/2.12.0.md).

## Push/Pull modes

VictoriaMetrics allows metrics data to be 'pushed' to the server in addition to it being 'pulled' by the server. When setting up services, you can decide which mode to use.

!!! alert alert-info "Note"

    For PMM 2.12.0 the default mode is 'pull'. Later releases will use the 'push' mode by default for newly-added services.

The mode (push/pull) is controlled by the `--metrics-mode` flag for the `pmm-admin config` and `pmm-admin add` commands.

If you need to change the metrics mode for an existing Service, you must remove it and re-add it with the same name and the required flags. (There is currently no ability to "update" a service.)

## Remapped targets for direct Prometheus paths

Direct Prometheus paths return structured information directly from Prometheus, bypassing the PMM application.

They are accessed by requesting a URL of the form `<PMM SERVER URL>/prometheus/<PATH>`.

As a result of the move to VictoriaMetrics some direct Prometheus paths are no longer available.

Here are their equivalents.

- `/prometheus/alerts` --> No change.
- `/prometheus/config` --> No equivalent. However, some information is at `/prometheus/targets`.
- `/prometheus/flags` --> The `flag` metrics at `/prometheus/metrics`.
- `/prometheus/graph` --> `/graph/explore` (Grafana) or `graph/d/prometheus-advanced/advanced-data-exploration` (PMM dashboard).
- `/prometheus/rules` --> No change.
- `/prometheus/service-discovery` --> No equivalent.
- `/prometheus/status` --> Some information at `/prometheus/metrics`. High cardinality metrics information at `/prometheus/api/v1/status/tsdb`.
- `/prometheus/targets` --> `/victoriametrics/targets`.

## Troubleshooting

To troubleshoot issues, see the VictoriaMetrics [troubleshooting documentation](https://victoriametrics.github.io/#troubleshooting).

You can also contact the VictoriaMetrics team via:

- [Google Groups](https://groups.google.com/forum/#!forum/victorametrics-users)
- [Slack](http://slack.victoriametrics.com/)
- [Reddit](https://www.reddit.com/r/VictoriaMetrics/)
- [Telegram](https://t.me/VictoriaMetrics_en)
