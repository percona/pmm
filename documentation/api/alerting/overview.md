---
title: Overview
slug: pmm-alert-thresholds
category:
  uri: alerting-api
position: 0
---

## Alert thresholds

An alert rule created from a template carries the threshold the template shipped with. Alert
threshold APIs let you change that threshold for one target — a single Node, for example —
without editing the template, duplicating the rule, or affecting anything else the rule
watches.

A rule whose template marks a parameter as overridable is registered with an identifier when
you create it, returned as `rule_id` in the [Create Alert Rule](ref:createrule) response. That
identifier is what the threshold endpoints address.

### The model

An **override** is a value set for one parameter of one rule on one target. A target is
identified by a **scope** and an id:

| Scope | Target is |
|---|---|
| `THRESHOLD_SCOPE_NODE` | a Node ID |
| `THRESHOLD_SCOPE_SERVICE` | a Service ID |
| `THRESHOLD_SCOPE_CLUSTER` | a cluster label value |

When more than one override could apply to the same series, the narrowest one wins:
`service`, then `node`, then `cluster`. A service runs on exactly one node, so a service
override is strictly narrower than a node override covering the same service.

> 🚧 Availability
> 
> Only `THRESHOLD_SCOPE_NODE` is currently supported. Service and cluster scopes are accepted by the schema but return `501 Not Implemented`. Omitting the scope means node.

Where no override applies, the rule evaluates against the template's default. Clearing an
override returns the target to that default — or to a broader override that still covers it.

### Endpoints

- [List Alert Thresholds](ref:listthresholds) reports the value each target is currently
  evaluated against, and whether it comes from an override or the default.
- [Set Alert Threshold](ref:setthreshold) overrides one parameter for one target.
- [Clear Alert Threshold](ref:clearthreshold) removes one override.
- [Batch Update Alert Thresholds](ref:batchupdatethresholds) applies several sets and clears
  in a single transaction.

### Permissions

These endpoints require **admin**. Unlike the rest of the alerting API, a viewer or editor
token cannot read or change thresholds.

To get the authentication token, check [Authentication](ref:authentication).
