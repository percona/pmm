# Manage the built-in alert rules

PMM creates alert rules that warn you when PMM itself has a problem, so you don't have to set them up yourself. Which rules you get depends on your deployment:

| Rule set | Created on | Folder |
|----------|------------|--------|
| [PMM internal components](templates_list.md#pmm_component_alerts) | Every PMM Server, including when you upgrade to PMM 3.10.0 | **PMM Server** |
| [PMM High Availability](templates_list.md#pmm_ha_alerts) | PMM HA Cluster only | **PMM High Availability** |
7890-[]
To see them, go to **Alerts > Alert rules** and open the folder for your rule set. The built-in rules are in the **PMM Managed** group.

## Get notified

To receive notifications from these rules:
{.power-number}

1. Configure a [contact point](contact_points.md).
2. In **Alerts > Alert rules**, check for rules you created earlier from the same templates. Delete them, or [turn off the built-in set](#turn-off-the-built-in-rules) to keep them, so you don't get notified twice. PMM never changes or removes rules you created.

## Customize the alert rules

You can't edit the built-in rules in the PMM UI. PMM manages them, so they always match your PMM version and any fixes reach you when you upgrade.

To change when an alert fires, for example its threshold, duration, or severity:
{.power-number}

1. [Create an alert rule](alert_rules.md#add-an-alert-rule-based-on-a-template) from the same template, with your own values.
2. [Turn off the built-in set](#turn-off-the-built-in-rules) so you don't get notified twice.

## Turn off the built-in rules

Each rule set has its own environment variable, so you can turn one off and keep the other:

| Variable | Default | Controls |
|----------|---------|----------|
| `PMM_ENABLE_COMPONENT_ALERTS` | `true` | Alerts when VictoriaMetrics, ClickHouse, Grafana, or the Query Analytics API stops responding. Applies to every PMM Server. |
| `PMM_ENABLE_HA_ALERTS` | `true` | Alerts about the cluster leader and unreachable nodes. Applies only to a PMM HA Cluster. |

To turn a set off, set its variable to `false` and restart PMM Server. The rules keep running until the restart, so to quiet one right away, [silence it](silence_alerts.md) instead.

Set the variable and restart PMM Server for your deployment:

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

After PMM Server restarts, the rules in that set no longer appear in **Alerts > Alert rules**. Rules you created, including copies you made from these templates, stay unchanged.

If you turn the set back on later, your silences still apply. Alerts that were firing start over and notify you again once their `for` duration has passed.

To remove both sets together with all Percona alert templates, you can instead [disable Percona Alerting](disable_alerts.md). This doesn't need a restart.
