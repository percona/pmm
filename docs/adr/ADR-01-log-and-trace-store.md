# ADR-01: Log and trace store

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590

## Context

PMM stores metrics in VictoriaMetrics and query analytics in ClickHouse. Logs and traces need a store as well. Three proofs of concept came before this decision:
- Loki ([PMM-9315](https://perconadev.atlassian.net/browse/PMM-9315));
- VictoriaLogs ([PMM-13391](https://perconadev.atlassian.net/browse/PMM-13391));
- OpenTelemetry with ClickHouse ([PMM-14169](https://perconadev.atlassian.net/browse/PMM-14169), draft PR #4230).

The engine evaluation is in [PMM-12768](https://perconadev.atlassian.net/browse/PMM-12768) (its presentation is linked from the ticket).

PMM-14169 lists the requirements drawn from the first two proofs of concept:
- stay away from AGPL-licensed software;
- reuse PMM's built-in components, ClickHouse and `grafana-clickhouse-datasource`;
- be HA-compatible;
- integrate with alerting;
- keep a simple change-deploy-test workflow.

## Options

1. **Loki.** It is licensed AGPL-3.0, which the requirements exclude.
2. **VictoriaLogs.** Apache-2.0. It is a separate storage engine with its own query language (LogsQL) and its own process, disk sizing and HA story, alongside ClickHouse.
3. **ClickHouse.** PMM already ships, sizes and runs it, including as a replicated cluster in PMM HA. The Grafana ClickHouse datasource is already provisioned and supports logs and traces. The OpenTelemetry Collector has an upstream ClickHouse exporter.

## Decision

Store logs and traces in ClickHouse, in a dedicated database `otel`.

## Consequences

- No new database and no AGPL component.
- One query language (SQL) across QAN, logs and traces.
- Logs and traces share ClickHouse's CPU, memory and disk with QAN. A size cap, a disk watermark and separate ClickHouse users with resource limits are required (ADR-05, ADR-04).
- The table layout is PMM's to define and to migrate (ADR-04).
- Raw SQL through a Grafana datasource can't be filtered per user, so access control needs its own design (ADR-07, ADR-18).
- The VictoriaLogs proof of concept's outcome is not recorded in PMM-13391. The reasons above come from PMM-14169 and from the 24 Sep 2026 review.
