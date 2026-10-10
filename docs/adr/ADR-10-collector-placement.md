# ADR-10: Collector placement

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15572

## Context

The collector has to run on every PMM client node and on PMM Server. The PoC in PR #4230 ran it as a standalone container. pmm-agent already supervises exporters as separate processes: it starts and restarts them, writes their config, reports their status, and passes them server credentials. pmm-client is installed as RPM, DEB, tarball or Docker image.

## Options

1. Embed the collector as a library inside pmm-agent.
2. Run it as a standalone container or service, outside pmm-agent.
3. Ship it as a binary that pmm-agent runs, as a new agent type.

## Decision

Option 3.

## Consequences

**For:**
- Lifecycle, config push, status, credentials and version gating follow the existing exporter path. Admins configure nothing on the node.
- It works on every install method, including hosts without a container runtime.
- pmm-agent's own dependency graph, crash domain and release cycle are unaffected by the collector's large dependency tree. A collector crash restarts only the collector.

**Against:**
- A new inventory agent type, with a minimum pmm-agent version. Older agents must be refused, because they would ignore the type without reporting an error.
- One more process per node.
