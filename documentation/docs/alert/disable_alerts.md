# Disable Percona Alerting

Percona Alerting is enabled by default in the PMM Settings. This feature adds the **Alert templates** option on the **Alerts** menu.

If for some reason you want to disable PMM Alert templates and keep only Grafana-managed alerts:
{.power-number}

1. Go to **Configuration > Settings > Advanced settings**.
2. Disable the **Percona Alerting** option. The **Alerts** menu will now display only Grafana-managed alert rules.
## Disable the built-in alert rules

PMM creates and maintains two sets of alert rules of its own: the [PMM component alerts](templates_list.md#pmm_component_alerts), on every server, and the [High Availability alerts](templates_list.md#pmm_ha_alerts), on a clustered one. Each has its own environment variable, so you can turn one off and keep the other:

| Variable | Default | Controls |
|----------|---------|----------|
| `PMM_ENABLE_COMPONENT_ALERTS` | `true` | The PMM component alerts, on every server |
| `PMM_ENABLE_HA_ALERTS` | `true` | The High Availability alerts, on a clustered server |

Set either to `false` and recreate the server to turn that set off. In Docker that means recreating the container with the variable set; with the Helm chart, change the value and upgrade the release.

Keep the following in mind:

- **These take effect when the server starts.** PMM writes the rules while it starts up, so a change needs the server recreated - which is what changing an environment variable requires anyway. To quieten a rule right now, without restarting anything, add a [silence](silence_alerts.md) instead.
- **Set the variable on every node.** In a High Availability deployment each node writes the rules for itself, so give all of them the same value.
- **Turning a set off deletes its rules.** Any silences you created keep working if you turn the set back on, because they match on labels and the labels do not change. The rules' own state does not survive, so anything that was firing is evaluated afresh and notifies again once its `for` duration has passed.
- **Your own rules are never touched.** Rules you created from these templates are yours, including copies you made to change a threshold.
- **Disabling Percona Alerting removes both sets** as well, since these rules are built from Percona alert templates. That switch is in **Configuration > Settings > Advanced settings**. It needs no server restart, but it is not instant either: PMM notices it on its next check, within a few minutes.
