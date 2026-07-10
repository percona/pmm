---
title: Configure PMM Server settings
slug: pmm-server-settings
category:
  uri: pmm-server-maintenance
position: 2
privacy:
  view: anyone_with_link
---

You can use the PMM Server Settings API to retrieve and update your PMM Server configuration programmatically. Use [Get settings](ref:getsettings) to check the current configuration, and [Change settings](ref:changesettings) to update it.

## Get current settings

```sh
curl -X GET https://<pmm-server-address>/v1/server/settings \
  -H "Authorization: Bearer glsa_xxxxx"
```

## Change settings

```sh
curl -X PUT https://<pmm-server-address>/v1/server/settings \
  -H "Authorization: Basic <base64-encoded-credentials>" \
  -H "Content-Type: application/json" \
  -d '{"data_retention": "2592000s"}'
```

## Field reference

### data_retention

Use this field to set how long PMM retains Prometheus and QAN data. You must express the value in seconds. If you use hours or minutes, the request will fail with an error.

| Format | Example | Supported |
|--------|---------|--------|
| Seconds | `2592000s` | ✔ |
| Minutes | `43200m` | ✘ |
| Hours | `720h` | ✘ |

Common values:

| Retention period | Value |
|-----------------|-------|
| 7 days | `604800s` |
| 30 days | `2592000s` |
| 90 days | `7776000s` |

### metrics_resolutions

Use this field to control how frequently PMM collects metrics. You can set high (`hr`), medium (`mr`), and low (`lr`) resolution intervals independently:

```json
{
  "metrics_resolutions": {
    "hr": "5s",
    "mr": "10s",
    "lr": "60s"
  }
}
```
