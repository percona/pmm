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

"Applies to it" means the parameter can be overridden at the scope you asked for. A
parameter its template restricts to service scope is omitted from a node-scoped listing,
because setting it there would be rejected — so every row you get back is one you can
actually write.

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

`scope` and `target` in the response describe where the **effective** value came from. Today
that is always the target you asked about, since node is the only scope that resolves. They
are reported separately because they will not always agree: once service and cluster scopes
are enabled, a node can inherit an override set on a cluster. Read them from the entry rather
than assuming the target you passed, and both are zero-valued — `THRESHOLD_SCOPE_UNSPECIFIED`
and `""` — when `is_overridden` is false.

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

> 🚧 Do not key a map on rule_id alone
> 
> With a target, `rule_id` and `param_name` together identify an entry. Without one, the response carries an entry per overridden target, so the same `rule_id` and `param_name` can repeat and differ only in `scope` and `target`. Key on all four.

### Reading zero values

Zero-valued fields **are** present. The gateway marshals with `EmitUnpopulated`, so a
threshold of `0` arrives as `"default_value": 0` rather than being omitted, and an entry that
is not overridden carries `"is_overridden": false`, `"scope": "THRESHOLD_SCOPE_UNSPECIFIED"`
and `"target": ""`. This is not the proto3 default of dropping zero values, so a client
written against that assumption will still work, but it need not treat an absent field as
`0`.
