---
title: Set and clear an alert threshold
slug: setting-alert-thresholds
category:
  uri: alerting-api
position: 2
---

## Set an alert threshold

Overrides one parameter of one rule for one target. The rule itself is not modified, and
every other target it watches keeps evaluating against the default.

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

The response returns the threshold as it now stands, in the same shape
[List Alert Thresholds](ref:listthresholds) uses.

Setting a threshold on a target that already has one replaces it. There is no separate
create-versus-update call.

### Validation

`value` must be finite and within the range the parameter declared:

| Condition | Status |
|---|---|
| Value outside the declared range, or not finite | `400 Bad Request` |
| Rule has no such overridable parameter | `404 Not Found` |
| Rule ID does not exist | `404 Not Found` |
| Target does not exist | `404 Not Found` |
| Parameter cannot be overridden at that scope | `400 Bad Request` |
| Scope is service or cluster | `501 Not Implemented` |

A parameter is only overridable if its template said so. A rule created before a template
gained an overridable parameter does not acquire one — the range and default are captured
when the rule is created, so an edit to the template afterwards does not change what an
existing rule validates against.

## Clear an alert threshold

Removes an override, returning the target to the rule's default or to a broader override that
still covers it.

```shell
curl --insecure -X DELETE \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/alerting/thresholds \
     --data '
{
  "scope": "THRESHOLD_SCOPE_NODE",
  "target": "dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d",
  "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d",
  "param_name": "threshold"
}
'
```

Clearing is idempotent: clearing a threshold that is not overridden succeeds and changes
nothing.

> 🚧 Clear rather than write the default back
> 
> To return a target to the default, clear the override — do not set the threshold to the default value. Writing the default as an override pins that target to today's value, so it will not follow a later change to the rule.

### Removing a target

Deleting a Node removes its overrides along with it. Cluster-scoped overrides are not
removed this way, because a cluster is a label value rather than an inventory entity and has
no removal event to hook.
