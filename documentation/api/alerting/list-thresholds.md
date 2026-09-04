---
title: List alert thresholds
slug: listing-alert-thresholds
category:
  uri: alerting-api
position: 1
---

## List alert thresholds

Reports the threshold each overridable parameter is currently evaluated against, and whether
that value comes from an override or from the rule's default.

The response depends on whether you name a target.

### For one target

Pass `scope` and `target` to get **every** overridable parameter that applies to it, whether
overridden or not. This is what a settings screen for a single Node needs — the untouched
parameters have to be shown alongside the changed ones.

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/alerting/thresholds?scope=THRESHOLD_SCOPE_NODE&target=dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d'
```

```json
{
  "thresholds": [
    {
      "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d",
      "param_name": "threshold",
      "summary": "A percentage from configured maximum",
      "unit": "PARAM_UNIT_PERCENTAGE",
      "default_value": 80,
      "effective_value": 95,
      "is_overridden": true,
      "scope": "THRESHOLD_SCOPE_NODE",
      "target": "dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d"
    }
  ]
}
```

`scope` and `target` in the response describe where the **effective** value came from, which
is not necessarily the target you asked about — a node can inherit a cluster-scoped override.
Both fields are absent when `is_overridden` is false.

### Across all targets

Omit `target` to get only the overrides that actually exist:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/alerting/thresholds'
```

Defaults are not enumerated here. Without a target there is no bounded set of targets to
enumerate them for — every Node PMM has ever monitored would qualify.

### Filtering by rule

`rule_id` narrows either form to a single rule:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/alerting/thresholds?rule_id=1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d'
```

> 🚧 Do not key a map on rule_id
> 
> Rules duplicated in Grafana share a `rule_id`, so two entries can carry the same `rule_id` and `param_name` and differ only in which rule they came from.

### Reading zero values

The API omits zero-valued fields, as JSON mapping for Protocol Buffers requires. A threshold
of `0` arrives with `default_value` or `effective_value` **absent**, not set to `0`, and an
entry that is not overridden has no `is_overridden` field at all. Treat an absent numeric
field as `0` rather than as unknown.
