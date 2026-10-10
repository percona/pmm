# ADR-14: OTLP export

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15391

## Context

Users ask for PMM to forward telemetry to other backends, and to ingest general OpenTelemetry metrics ([PMM-15391](https://perconadev.atlassian.net/browse/PMM-15391)).

## Options

1. Include OTLP export and OTel metrics ingestion in PMM-15567.
2. Leave both out.

## Decision

Option 2. Exporting logs, traces or metrics to external backends, and general OTel metrics ingestion, are out of scope for PMM-15567.

## Consequences

- The OCB build has no exporters except `otlphttp`, used for node-to-server traffic, and `clickhouse` (ADR-09).
- PMM-15391 can add exporters later as a deliberate manifest change.
