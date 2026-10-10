# ADR-02: Identity contract

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15582

## Context

Logs and traces must join PMM inventory, metrics and QAN by service and time. The prototype used three different key sets:
- PMM label names (`node_id`, `agent_id`) on logs;
- `pmm.node_id` in the eBPF contract;
- `node_name` and `pmm_source` in other code.

OpenTelemetry defines `service.name` as the application that emits the telemetry.

## Options

1. Reuse PMM label names (`node_id`, `service_name`) as resource attributes.
2. Map PMM identity onto OTel semantic conventions only (`service.name` = PMM service).
3. Use a `pmm.` namespace for PMM identity, and keep `service.name` with its OTel meaning.

## Decision

Option 3. Every record carries these resource attributes:

| Attribute | When | Value |
|---|---|---|
| `pmm.node_id` | always | the node of the collector that read or received the record |
| `pmm.agent_id` | always | the collector agent's id |
| `pmm.service_id`, `pmm.service_name` | record belongs to a PMM service | the PMM service |
| `service.name` | application sources, PMM Server components, OTLP senders | the application, never the PMM service |

Log-source configuration (path, preset, labels) is never copied onto records.

## Consequences

- Records join inventory by `pmm.service_id` and `pmm.node_id`, whatever the service is called.
- Applications that already emit OTLP keep their own `service.name`, and need no PMM-specific setup.
- The node collector sets the `pmm.` attributes, and overwrites values a sender supplied (PMM-15579). How the server verifies them is in ADR-18.
