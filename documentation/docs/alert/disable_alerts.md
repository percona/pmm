# Disable Percona Alerting

Percona Alerting is enabled by default in the PMM Settings. This feature adds the **Alert templates** option on the **Alerts** menu.

If for some reason you want to disable PMM Alert templates and keep only Grafana-managed alerts:
{.power-number}

1. Go to **Configuration > Settings > Advanced settings**.
2. Disable the **Percona Alerting** option. The **Alerts** menu will now display only Grafana-managed alert rules.

This also removes the [built-in alert rules](built_in_alerts.md). The change doesn't need a restart, but PMM can take a few minutes to apply it.
