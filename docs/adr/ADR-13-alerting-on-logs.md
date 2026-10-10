# ADR-13: Alerting on logs

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-14689

## Context

Users will want alerts on log patterns, such as deadlocks or replication errors. PMM alerting today evaluates PromQL in VMAlert. Log alerting would need ClickHouse-backed rules, rate limits, and a decision on where the rules are evaluated.

## Options

1. Include alerting on logs in PMM-15567.
2. Leave it to a follow-up epic.

## Decision

Option 2. Alerting on logs is out of scope for PMM-15567, and is tracked in [PMM-14689](https://perconadev.atlassian.net/browse/PMM-14689).

## Consequences

- The first release covers storage, parsing, access and visualisation.
- Alert templates for the collectors' own health (dropped records, full queues, paused ingest) are in scope (PMM-15593), because they use metrics, not logs.
