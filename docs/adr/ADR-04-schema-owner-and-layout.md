# ADR-04: Schema owner and table layout

- Status: Proposed. The schema owner was decided on 2026-09-24. The sort key and the `service.name` value for database sources were chosen on 2026-10-10, and wait for architect confirmation.
- Date: 2026-10-10
- Tickets: PMM-15567, PMM-15590, PMM-15571

## Context

- **Who creates the schema.** The upstream ClickHouse exporter can create its own tables (`create_schema: true`), but then nothing versions them, makes them cluster-aware, or migrates them. QAN's schema is owned by qan-api2's versioned migrations, which handle `Replicated` engines on clusters.
- **Sort key.** A table's sort key can't change after the first release.
- **Column naming.** The exporter fills its `ServiceName` column from `service.name`, which ADR-02 defines as the application, not the PMM service.
- **Migration chain.** qan-api2 migrations are bound to one database (`pmm`) with one `schema_migrations` table.

## Options

**Owner:**
1. exporter `create_schema`;
2. pmm-managed;
3. qan-api2 migrations.

**Sort key:**
1. the upstream key, `(ServiceName, TimestampTime)`;
2. time first;
3. PMM node and service first.

## Decision

- **Owner:** qan-api2 owns the `otel` schema, in a separate embedded migration chain (`qan-api2/migrations/otel/sql/`) with its own migrations table, `otel_schema_migrations`. The collector runs with `create_schema: false`.
- **Columns:** keep the exporter's column set unchanged, and add two materialized columns:
  - `PmmNodeId LowCardinality(String) MATERIALIZED ResourceAttributes['pmm.node_id']`
  - `PmmServiceId LowCardinality(String) MATERIALIZED ResourceAttributes['pmm.service_id']`
- **Layout:** `PARTITION BY toDate(TimestampTime)`, `ORDER BY (PmmNodeId, PmmServiceId, ServiceName, TimestampTime)`. There is no table TTL (ADR-05).
- **`service.name` for database sources:** the server program (`mysqld`, `postgres`, `mongod`).

## Consequences

- The exporter and the Grafana plugin work against the table unchanged.
- Filtering by PMM node and service, the common case and the shape LBAC needs (ADR-18), reads only the matching ranges.
- A query for "everything in the last five minutes, across all nodes" reads more granules than with a time-first key. This is acceptable for a per-node and per-service product.
- A CI contract test runs the pinned collector against the migrated schema, so a collector upgrade can't break inserts unnoticed.
