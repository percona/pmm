---
title: Set and clear a threshold override
slug: setting-alert-thresholds
category:
  uri: alerting-api
position: 2
---

## Set a threshold override

Sets a per-target override on one parameter of an alert rule. Only alert rules created from a [Dynamic template](https://docs.percona.com/percona-monitoring-and-management/3/alert/alert-thresholds.html) support overrides. 

To set the override, pass the `rule_id` from when you created the rule, the node ID, and the new threshold value:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/alerting/thresholds \
     --data '
{
  "scope": "THRESHOLD_SCOPE_NODE",
  "target": "dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d",
  "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d",
  "param_name": "threshold",
  "value": 95
}
'
```

The response returns the updated threshold in the same shape as [List alert thresholds](ref:listthresholds). If the alert fires on this target, the alert text reports the effective threshold, not the rule's default.

### How overrides are applied

If an override already exists for this parameter on this target, the new value replaces it. The rule itself is not changed, and every other target it watches keeps evaluating against the default.

### Rules and templates

A rule's overridable parameters, their ranges, and defaults are captured when the rule is created. Editing a template after creating a rule does not update the rule. To enable overrides on an existing rule, recreate it from the updated template.

### Error codes

The request fails in any of the following cases:

| Condition | Status |
|---|---|
| Value outside the declared range, or not finite | `400 Bad Request` |
| Rule has no such overridable parameter | `404 Not Found` |
| Rule ID does not exist | `404 Not Found` |
| Target does not exist | `404 Not Found` |
| Parameter cannot be overridden at that scope | `400 Bad Request` |
| Scope is service or cluster (currently not supported) | `501 Not Implemented` |

## Clear a threshold override

Removes an override and returns the target to the rule's default, or to a broader override that still covers it.

To clear the override, pass the same `rule_id`, target, and parameter name you used to set it:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/alerting/thresholds:clear \
     --data '
{
  "scope": "THRESHOLD_SCOPE_NODE",
  "target": "dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d",
  "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d",
  "param_name": "threshold"
}
'
```

If no override exists for that parameter on that target, the request succeeds and changes nothing. The rule ID, parameter name, and target are still validated, so an unknown value for any of them returns an error.

### Remove overrides

#### Reset a target to the default

To return a target to the rule's default, clear the override.

> 🚧 Do not set the threshold to the default value
>
> Setting the default value keeps the override in place, so the target will not pick up future changes to the rule.

#### When you delete a node

If you delete a node, its overrides are removed along with it. Cluster-scoped overrides are not removed this way, because clusters are label values rather than inventory items and have no deletion event to trigger a cleanup.

#### When you delete a rule

If you delete an alert rule in Grafana, you may still see its overrides when you list thresholds for a target. They are stale and harmless (no rule evaluates them), and you do not need to remove them manually. 

PMM cleans them up the next time you write a threshold (a set, clear, or batch update), at most once every 15 minutes, once it confirms the rule is gone from Grafana.
