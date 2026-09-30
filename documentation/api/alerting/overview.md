---
title: Overview
slug: pmm-alert-thresholds
category:
  uri: alerting-api
position: 0
---

## Alert thresholds

An alert rule created from a template compares every target it watches against the same threshold. Alert threshold APIs let you change that threshold for one target — a single Node, for example — without editing the template, duplicating the rule, or affecting anything else the rule watches.

A rule whose template marks a parameter as overridable is registered with an identifier when you create it, returned as `rule_id` in the [Create Alert Rule](ref:createrule) response. That identifier is what the threshold endpoints address.

A rule created from a template with no overridable parameters gets an empty `rule_id`, since there is nothing for a threshold to be keyed on. Rules created before PMM 3.10.0 have no `rule_id` either; recreate them from the template to make their thresholds overridable.

### Quick start

Create a rule from an overridable template, such as the built-in `pmm_node_high_cpu_load`, and note the `rule_id` in the response:

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

```json
{
  "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d"
}
```

The `80` you pass becomes the rule's default for every Node. Then raise the threshold for one Node with [Set Alert Threshold](ref:setthreshold), and check the result with [List Alert Thresholds](ref:listthresholds).

### Finding a rule's ID

The `rule_id` is also stored on the Grafana rule as the `pmm_rule_id` label. For a rule you did not create yourself, read the label from the rule in the Grafana UI or from the Grafana alerting API.

### The model

An **override** is a value set for one parameter of one rule on one target. A target is identified by a **scope** and an id:

| Scope | Target is |
|---|---|
| `THRESHOLD_SCOPE_NODE` | a Node ID |
| `THRESHOLD_SCOPE_SERVICE` | a Service ID |
| `THRESHOLD_SCOPE_CLUSTER` | a cluster label value |

When more than one override could apply to the same series, the narrowest one wins: `service`, then `node`, then `cluster`. A service runs on exactly one node, so a service override is strictly narrower than a node override covering the same service.

> 🚧 Availability
> 
> Only `THRESHOLD_SCOPE_NODE` is currently supported. Service and cluster scopes are accepted by the schema but return `501 Not Implemented`. Omitting the scope means node.

Where no override applies, the rule evaluates against its default: the value supplied for the parameter when the rule was created, not the template's value. Clearing an override returns the target to that default — or to a broader override that still covers it.

### When a change takes effect

A write is stored immediately, and the rule picks it up at its next evaluation after PMM Server's next internal metrics scrape. With the default settings, that is within about a minute: the scrape runs at the medium metrics resolution (10 seconds by default) and PMM-created rule groups evaluate every minute. An alert still has to meet the rule's `for` duration before it fires or resolves.

If PMM Server cannot provide the overrides — for example while it restarts — rules keep using the last known overrides for up to 5 minutes. After that, overridden targets fall back to the default until the overrides are available again.

### Editing and copying rules in Grafana

The default and the allowed range are captured when the rule is created. Nothing reads them back from Grafana, so:

- Editing the threshold inside the rule's query in Grafana does not change the `default_value` these endpoints report.
- A copy of the rule keeps the `pmm_rule_id` label, so the copy and the original share the same overrides.
- Removing the `pmm_rule_id` label detaches the rule; its overrides are removed as described in [Set and clear an alert threshold](ref:setting-alert-thresholds).

### Endpoints

- [List Alert Thresholds](ref:listthresholds) reports the value each target is currently evaluated against, and whether it comes from an override or the default.
- [Set Alert Threshold](ref:setthreshold) overrides one parameter for one target.
- [Clear Alert Threshold](ref:clearthreshold) removes one override.
- [Batch Update Alert Thresholds](ref:batchupdatethresholds) applies several sets and clears in a single transaction.

### Permissions

These endpoints require the **Admin** role in the main organization (ID `1`), or Grafana server admin. Unlike the rest of the alerting API, a viewer or editor token cannot read or change thresholds. Percona Alerting must be enabled; while it is disabled, the endpoints return an error.

To get the authentication token, check [Authentication](ref:authentication).
