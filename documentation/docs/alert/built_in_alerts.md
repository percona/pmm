# Manage the built-in alert rules

PMM includes preconfigured alert rules that watch PMM's own components and fire when one of them stops working. The rules are active by default, but you need a [contact point](contact_points.md) configured to receive the notifications.

| Rule set | Included with | Monitors |
|----------|---------------|----------|
| [PMM internal components](templates_list.md#pmm_component_alerts) | Every PMM Server | VictoriaMetrics, ClickHouse, Grafana, and the Query Analytics API |
| [PMM High Availability](templates_list.md#pmm_ha_alerts) | PMM HA Cluster deployments only | The cluster leader and node availability |

To view the rules PMM created for your deployment, go to **Alerts > Alert rules** and open the folder for your rule set: **PMM Server** for the internal components rules, or **PMM High Availability** for the HA rules.

## Get notified

The built-in rules fire even if no one receives their alerts. To get notified, [configure a contact point](contact_points.md).

## Customize the alert rules

To adjust a threshold, duration, or severity, create your own rule from the same template with the values you want, then [turn off the built-in set](#turn-off-the-built-in-rules) so you don't get notified twice. See [Add an alert rule based on a template](alert_rules.md#add-an-alert-rule-based-on-a-template).

## Turn off the built-in rules

Turn off a rule set if you've replaced it with your own rules, or if you don't want those alerts at all. Each set has its own environment variable, so you can turn one off and keep the other:

| Variable | Default | Turns off alerts for | Applies to |
|----------|---------|----------------------|------------|
| `PMM_ENABLE_COMPONENT_ALERTS` | `true` | VictoriaMetrics, ClickHouse, Grafana, and the Query Analytics API | Every PMM Server |
| `PMM_ENABLE_HA_ALERTS` | `true` | The cluster leader and node availability | PMM HA Clusters only |

The rules keep firing until PMM Server restarts. If you need to suppress notifications before that, [silence the rules](silence_alerts.md). Silencing mutes notifications without stopping the rules from firing.

Set the variable to `false` for your deployment:

=== "Docker"

    To recreate the container with the variable set:
    {.power-number}

    1. Stop the PMM Server container and rename it:

        ```sh
        docker stop pmm-server
        docker rename pmm-server pmm-server-old
        ```

    2. Start a new container from the old one's data, with the variable set to `false`. Use the same options you used when you first started PMM Server, for example:

        ```sh
        docker run \
        --detach \
        --restart always \
        --publish 443:8443 \
        --volumes-from pmm-server-old \
        --env PMM_ENABLE_COMPONENT_ALERTS=false \
        --name pmm-server \
        percona/pmm-server:3
        ```

=== "Helm"

    To update your release with the variable set:
    {.power-number}

    1. Add the variable to `pmmEnv` in your `values.yaml` file:

        ```yaml
        pmmEnv:
          PMM_ENABLE_COMPONENT_ALERTS: "false"
        ```

    2. Apply the change to your release:

        ```sh
        helm upgrade <release> <chart> -n <namespace> -f values.yaml
        ```

        For example, `helm upgrade pmm-ha percona/pmm-ha -n pmm -f values.yaml` for a PMM HA Cluster.

## What happens after you disable a rule set

After PMM Server restarts, the rules in that set no longer appear in **Alerts > Alert rules**. Rules you created, including copies you made from these templates, stay unchanged.

If you turn the set back on later, any silences you set still apply. Alerts that were firing before you turned the set off start over and notify you again once their `for` duration has passed.

To remove both sets together with all Percona alert templates, [disable Percona Alerting](disable_alerts.md) instead. This takes effect immediately without a restart.
