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

To use the threshold override endpoints, you need:

- An [Admin-role token](ref:authentication). Viewer and editor tokens return `403`.
- Percona Alerting enabled in **Configuration > Settings**. Requests fail if it is disabled.
- A rule created from a **Dynamic** template. Dynamic templates have at least one parameter with `overridable: true` and show a **Dynamic** badge in **Alerts > Templates**. You can also check for `overridable: true` in the `GET /v1/alerting/templates` response. The built-in `pmm_node_high_cpu_load` template is Dynamic. When you create the rule, the response includes a `rule_id`. That is the identifier the threshold endpoints use.

### Enable overrides for older rules

Rules created before PMM 3.10.0 don't have a `rule_id` and don't support overrides. To get one:

1. Check **Alerts > Templates** for a **Dynamic** badge on your template. The built-in `pmm_node_high_cpu_load` template already has one. If yours uses `expr`, it doesn't support overrides. Rewrite it to use `queries`, `expressions`, and `condition` with `overridable: true`.
2. Delete the rule.
3. Recreate it from the template.

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

PMM records the rule's default threshold and allowed range when you create it and doesn't update them if you later modify the rule in Grafana.

Editing the threshold inside the rule in Grafana has no effect on the `default_value` these endpoints report. It stays fixed at the value you set when you created the rule.

If you copy a rule, the copy inherits the `pmm_rule_id` label and shares the same overrides as the original. To manage overrides independently, create a new rule from the template instead.

Don't remove the `pmm_rule_id` label. PMM uses it to tell that the rule still exists. Without it, PMM eventually deletes the rule's overrides and the rule falls back to its default threshold.

### Endpoints

Use these endpoints to manage threshold overrides:

- [List alert thresholds](ref:listthresholds) to see the threshold each node is currently evaluated against and whether it comes from an override or the rule's default.
- [Set alert threshold](ref:setthreshold) to override the threshold for one node.
- [Clear alert threshold](ref:clearthreshold) to remove an override and return the node to the rule's default.
- [Batch update alert thresholds](ref:batchupdatethresholds) to apply several changes in a single transaction.
