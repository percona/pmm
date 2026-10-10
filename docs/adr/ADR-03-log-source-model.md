# ADR-03: Log-source model

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15573

## Context

PMM needs to collect three kinds of logs: database logs, operating-system logs, and logs of applications that are not PMM services. Each kind must carry the right identity (ADR-02), and each must be addable with one command.

## Options

1. Every log source must belong to a PMM service, so applications would be registered as services.
2. Every source belongs to the node, and services are attached as labels.
3. Every source belongs to the node's single collector, with one of three owners.

## Decision

Option 3. A log source is a file (a path or a glob) or a journald unit, attached to the node's only OTEL collector. Its owner is one of:

| Owner | Use | Identity on each row | Added with |
|---|---|---|---|
| A PMM service on this node | database logs | `pmm.node_id`, `pmm.service_id`, `pmm.service_name` | `pmm-admin add <db>` (ADR-19), or `pmm-admin add logs --service-name` |
| The node | OS logs | `pmm.node_id` | `pmm-admin add logs --path` or `--journald` |
| The node plus an app name | application logs | `pmm.node_id`, `service.name` | `pmm-admin add logs --path --app` |

Further rules:
- Sources live in their own table, not in agent labels.
- A source bound to a service on another node is rejected.
- Removing a service removes its sources.
- Applications are not added as PMM services. Applications that send OTLP need no source.

## Consequences

- Each node has one collector, created on first use, so one process and one config per node.
- Remote and RDS instances have no log sources: the collector reads files on its own host only.
- The UI and pmm-admin share one API for sources (PMM-15573, PMM-15574).
