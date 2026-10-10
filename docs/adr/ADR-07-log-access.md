# ADR-07: Who can read logs

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15576, PMM-15591

## Context

Logs can contain SQL text, personal data and secrets. PMM's label-based access control (LBAC) works by sending each user's selectors to components that filter on them. A Grafana ClickHouse datasource takes raw SQL, so it can't be filtered that way. Grafana OSS has no per-datasource permissions. The nginx auth subrequest doesn't receive the request body, so it can't see which datasource a `/graph/api/ds/query` request targets.

## Options

1. Every Grafana user can read logs.
2. Only Admins can read logs, until LBAC covers them.
3. Ship LBAC for logs from the start.

## Decision

Option 2:
- **Grafana:** a plugin-client middleware in Percona's Grafana fork denies queries to the OTEL datasource (uid `pmm-otel`) from anyone whose org role isn't Admin.
- **ClickHouse:** the datasource connects as `otel_reader`, which can only `SELECT` on `otel.*`.
- **Redaction:** before storage, the server collector masks bearer tokens, `glsa_` tokens, `Authorization` headers and `password=` values in log bodies and attributes.
- **LBAC:** it comes later (ADR-18).

## Consequences

- Viewers, Editors and LBAC-restricted users see no logs in this release. That is fail-closed, and matches how logs.zip is treated today.
- Redaction is pattern-based. Unknown secret formats and personal data are not masked, which is why access stays Admin-only.
- The Admin check lives in the Grafana fork, so the fork needs a matching change in each release.
