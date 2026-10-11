# ADR-21: Native log explorer in the PMM UI

- Status: Proposed
- Date: 2026-10-11
- Tickets: PMM-15567, PMM-15576, PMM-15722

## Context

PMM-15576 gives users a Grafana ClickHouse datasource and a Logs dashboard. That has three limits:
- **Access control:** raw SQL from a datasource can't be filtered per user (ADR-07, ADR-18).
- **Names:** dashboards get node and service names only through workarounds, because records carry IDs (ADR-02).
- **Log reading:** common log tasks are awkward in a dashboard: surrounding lines, live tail, a details view, links to QAN and dashboards.

The PMM UI already has native data pages built with `@percona/peak-ui` (for example Real-Time Analytics). peak-ui provides a data table, chips, code blocks, autocomplete and date-time inputs, details-pane navigation, URL-synced table state, and illustrations. It has no chart components.

## Options

1. Grafana only: a datasource plus dashboards.
2. A native explorer that queries ClickHouse directly from the browser through Grafana's datasource proxy.
3. A native explorer on a PMM API that builds the SQL from structured filters.

## Decision

Option 3:
- **API:** pmm-managed exposes `LogsService` (`Search`, `Context`, `Histogram`, `Facets`) under `/v1/logs`.
  - It reads `otel.logs` as `otel_reader`, and binds every filter value as a parameter.
  - It caps the time range, result size and execution time.
  - It resolves node and service names from inventory.
- **Page:** a PMM UI page, `/logs`, built with peak-ui and TanStack Query, offers filters with counts, a severity histogram, a table, a details pane with surrounding lines, live tail, export, and shareable URLs.
- **Links:** service dashboards and Inventory link into it.
- **Access:** Admins only until ADR-18 is implemented. Then the same API applies LBAC and the page opens to every role.
- **Grafana:** the datasource and dashboard are built too, Admin-only. Which of the two log UIs to keep is decided after Technical Preview feedback; until then, service dashboards link to the explorer.

## Consequences

- One read path that PMM controls: the place where LBAC (ADR-18) is enforced later, with no SQL from clients.
- **Two ways to read logs to maintain:** the explorer is the primary one, and the dashboard is secondary.
- **The histogram uses `@mui/x-charts`** (MIT, built for MUI v7). It is a new UI dependency, and later charts, such as the trace view, reuse it.
- New UI work in a release that also ships the collector. If it can't ship with 3.11.0, logs are available through Grafana first.
