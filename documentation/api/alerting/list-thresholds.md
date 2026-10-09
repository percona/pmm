---
title: List threshold overrides
slug: listing-threshold-overrides
category:
  uri: alerting-api
position: 1
---

## List threshold overrides

Use this endpoint to check what threshold a node is currently being evaluated against and whether it comes from an override or the rule's default.

### View thresholds for a specific node

To see every overridable parameter for one node, pass `scope` and `target`. The response includes both overridden and non-overridden parameters, so you can see the full picture for that node:

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

`effective_value` is the threshold this node is currently evaluated against. If `is_overridden` is `true`, that value comes from an override you set. If `is_overridden` is `false`, the node is using `default_value`, the threshold you set when you created the rule.

Every parameter in the response is one you can override for this node. If a parameter does not appear, you cannot set an override for it at this scope.

### See all active overrides

To see every override currently set across all nodes, omit `target`:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/alerting/thresholds'
```

Nodes using the rule's default are not included. Without a target, there is no bounded set of nodes to enumerate defaults for.

### Filter by rule

To see overrides for one rule only, add `rule_id`:

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --url 'https://127.0.0.1/v1/alerting/thresholds?rule_id=1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d'
```

> 🚧 The same rule and parameter can appear more than once
>
> If you query without a target, the same rule and parameter can appear multiple times, once for each node that has that override set. Check `scope` and `target` to tell the entries apart.

### Reading the response

`scope` and `target` in each entry describe where the effective value came from. Currently this always matches the target you passed, since node is the only supported scope. 

Read them from the entry rather than assuming they match your request. Once service and cluster scopes are enabled, a node can inherit an override set at a broader scope and the values will differ. 

When `is_overridden` is false, both fields are empty (`THRESHOLD_SCOPE_UNSPECIFIED` and `""`).

Zero-valued fields are always present. A threshold of `0` appears as `"default_value": 0` rather than being omitted, and a non-overridden entry carries `"is_overridden": false` rather than omitting the field.
