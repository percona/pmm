---
title: Batch update alert thresholds
slug: batch-updating-alert-thresholds
category:
  uri: alerting-api
position: 3
---

## Batch update alert thresholds

Applies several threshold changes in a **single transaction**: either every update lands or
none does.

This is what a form editing several rows at once should use. Issuing the changes as separate
[Set](ref:setthreshold) and [Clear](ref:clearthreshold) calls risks a partial result that the
client cannot report coherently — some rows saved, one rejected, and no way to tell the user
which state the system is now in.

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

### Setting and clearing in one call

Whether an entry sets or clears is decided by `value`:

- **`value` present** — sets the override to that value.
- **`value` omitted** — clears the override.

The second entry in the example above clears its threshold, because it has no `value`.

> 🚧 Omit the field, do not send zero
> 
> `value` is optional precisely so that omitting it can mean *clear*. Sending `"value": 0` sets the threshold to zero, which is a real and very different instruction.

### Response

The response lists the thresholds that are now overridden. **Cleared entries are omitted** —
after a successful clear there is no override to report, so a request of three sets and two
clears returns three thresholds.

At least one update is required. Validation is the same as for
[Set Alert Threshold](ref:setthreshold), applied to every entry; one invalid entry rolls the
whole batch back and nothing is written.
