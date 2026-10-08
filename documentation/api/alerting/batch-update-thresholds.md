---
title: Batch update threshold overrides
slug: batch-updating-threshold-overrides
category:
  uri: alerting-api
position: 3
---

## Batch update the threshold overrides

Use this endpoint to set or clear several threshold overrides in one call. Either all changes are applied or none are, so you never end up with a partial result.

Use this instead of separate [Set](ref:setthreshold) and [Clear](ref:clearthreshold) calls when you need to update multiple nodes or parameters at once. If one entry is invalid, the whole batch is rejected and nothing is written.

To send the batch, list all your changes in the `updates` array:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/alerting/thresholds:batchUpdate \
     --data '
{
  "updates": [
    {
      "scope": "THRESHOLD_SCOPE_NODE",
      "target": "dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d",
      "rule_id": "1f8b2c34-5d6e-4a7b-8c9d-0e1f2a3b4c5d",
      "param_name": "threshold",
      "value": 95
    },
    {
      "scope": "THRESHOLD_SCOPE_NODE",
      "target": "dc1f7e40-1b1a-4c5d-9f2e-2b6a1e3f4c5d",
      "rule_id": "2a9c3d45-6e7f-4b8c-9d0e-1f2a3b4c5d6e",
      "param_name": "threshold"
    }
  ]
}
'
```

### Set and clear in one call

Each entry either overrides a threshold or returns a node to the rule's default. To override, include `value`. To return a node to the default, leave `value` out. The second entry in the example above returns its node to the default because it has no `value`.

> 🚧 Omit `value` to clear, do not send zero
>
> Sending `"value": 0` sets the threshold to zero, which is a real override. Omit the field entirely to clear the override.

### Response

The response lists the overrides that are now active. Cleared entries are not included. After a successful clear there is no override to report, so a batch of three sets and two clears returns three entries.

### Error codes

The batch must include at least one entry. If any entry fails validation, none of your changes are applied. You get a clean failure with no partial updates. The same rules apply as for [Set a threshold override](ref:setthreshold). Entries that remove an override that does not exist succeed and change nothing.
