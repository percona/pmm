# ADR-17: Profiles are deferred

- Status: Proposed
- Date: 2026-10-10
- Tickets: PMM-15567

## Context

OpenTelemetry's fourth signal, continuous profiling, entered public Alpha in March 2026. The contrib ClickHouse exporter's README lists its stability as "development" for profiles, "alpha" for metrics, and "beta" for traces and logs. PMM-15567 doesn't mention profiles. A request to make the receiver "ready for profiles" raised the question of whether to plan for them now.

## Options

1. Plan profiles storage and a pipeline now.
2. Run a time-boxed spike, with nothing merged.
3. Defer, and record why.

## Decision

Option 3. PMM-15567 includes no profiles pipeline, table or UI. Revisit when the ClickHouse exporter marks profiles beta, or when a stable ClickHouse schema for OTLP profiles exists.

## Consequences

- No schema is fixed for an unstable signal, so no migrations need undoing later.
- The OCB manifest (ADR-09) needs no profiles components.
- Users who need profiling now use a separate tool.
