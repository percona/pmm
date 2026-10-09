---
title: Alert threshold overrides
slug: pmm-alert-threshold-overrides
category:
  uri: alerting-api
position: 0
---

## Alert threshold overrides

Use these endpoints to override alert thresholds for individual nodes without changing the rule or template.

### Requirements

To use these endpoints, you need:

- The **Admin** role. Viewer and editor tokens return `403`.
- Percona Alerting enabled in **Configuration > Settings**. If it is off, every request returns an error.
- A rule created from a **Dynamic** template. Dynamic templates have at least one parameter with `overridable: true`. You can identify them by the **Dynamic** badge in **Alerts > Templates**, or by checking for `overridable: true` in the `GET /v1/alerting/templates` response. The built-in `pmm_node_high_cpu_load` template is Dynamic.

When you create a rule from a Dynamic template, the response includes a `rule_id`. That identifier is what the threshold endpoints use to address the rule. Rules created before PMM 3.10.0 were not assigned a `rule_id` because threshold overrides did not exist yet. To enable overrides on such a rule, delete it and create a new one from the same template.

To get an authentication token, see [Authentication](ref:authentication).

### Get started

To override the alert threshold for one node without affecting the others:
{.power-number}

1. Create a rule from a Dynamic template, such as the built-in `pmm_node_high_cpu_load`. The `80` you pass becomes the rule's default for every node:

    ```shell
    curl --insecure -X POST \
         --header 'Authorization: Bearer XXXXX' \
         --header 'Content-Type: application/json' \
         --url https://127.0.0.1/v1/alerting/rules \
         --data '
    {
      "template_name": "pmm_node_high_cpu_load",
      "name": "Node high CPU load",
      "folder_uid": "XXXXX",
      "group": "default-alert-group",
      "severity": "SEVERITY_WARNING",
      "for": "300s",
      "params": [{"name": "threshold", "type": "PARAM_TYPE_FLOAT", "float": 80}]
    }
    '
    ```

2. Save the `rule_id` from the response. If you didn't create the rule yourself, find the `rule_id` stored on the Grafana rule as the `pmm_rule_id` label:

    ```json
    {
      "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d"
    }
    ```

3. [Set Alert Threshold](ref:setthreshold) to raise or lower the threshold for one node.

4. [List Alert Thresholds](ref:listthresholds) to confirm the override is in effect.

### How overrides work

When you set an override, you apply it to one target at a time. Identify the target by its scope and ID:

| Scope | Target |
|---|---|
| `THRESHOLD_SCOPE_NODE` | a Node ID |
| `THRESHOLD_SCOPE_SERVICE` | a Service ID |
| `THRESHOLD_SCOPE_CLUSTER` | a cluster label value |

If more than one override could apply to the same node, the narrowest scope wins: service beats node, node beats cluster.

> 🚧 Node scope only
>
> You can currently set overrides for nodes only. Using service or cluster scope returns `501 Not Implemented`. If you omit the scope, PMM defaults to node.

When no override applies, the rule uses the threshold you set when you created it, not the template's value. Clearing an override returns the node to that default.

### When a change takes effect

Changes are stored immediately, but the rule picks them up at its next evaluation after PMM's internal metrics scrape. With default settings, that is within about a minute. The alert still has to meet the rule's `for` duration before it fires or resolves.

If PMM Server is unavailable, rules keep using the last known overrides for up to 5 minutes. After that, nodes with overrides fall back to the rule's default.

### Editing, copying, and modifying rules in Grafana

PMM captures the rule's default and allowed range at creation time and doesn't sync with Grafana after that. Be aware of these behaviors when working with rules directly in Grafana.

If you edit the threshold directly inside the rule in Grafana, the `default_value` these endpoints report won't change. It's fixed at the value you set when you created the rule.

If you copy a rule in Grafana, the copy shares the same overrides as the original because it keeps the `pmm_rule_id` label. To manage overrides independently, create a new rule from the template instead.

If you remove the `pmm_rule_id` label from a rule, the rule is detached from the threshold system and its overrides are deleted. See [Set and clear an alert threshold](ref:setting-alert-thresholds).

### Endpoints

Use these endpoints to manage threshold overrides:

- [List alert thresholds](ref:listthresholds) to see the threshold each node is currently evaluated against and whether it comes from an override or the rule's default.
- [Set alert threshold](ref:setthreshold) to override the threshold for one node.
- [Clear alert threshold](ref:clearthreshold) to remove an override and return the node to the rule's default.
- [Batch update alert thresholds](ref:batchupdatethresholds) to apply several changes in a single transaction.
