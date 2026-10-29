# Proxy and HA correlation (Phase 1)

## Proxies (traffic path)

Model HAProxy, ProxySQL, pgBouncer, and mongos as **nodes** or **annotated edges** using `pmm.component_role=proxy` (or `router`) plus stable `service.name` / peer attributes. Deep protocol parsing is Phase 2.

## Orchestrators (control plane)

Patroni (and equivalents) provide **failover windows** and role-change markers attached to database-side nodes or edges (`AnnotateFailoverWindow` in `managed/otel/correlation.go`). MVP depth: coarse time ranges + text note, not full DCS event stream.

## Stitching

User journeys: **app → proxy → database**, with optional failover shading when orchestrator annotations overlap the selected time range.
