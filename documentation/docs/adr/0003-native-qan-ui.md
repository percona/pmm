# ADR-003: Native Query Analytics (QAN) UI

## Status

Accepted.

## Context

Historical Query Analytics is implemented as a Grafana dashboard plugin (`pmm-qan-app-panel`). PMM is migrating primary workflows to the native UI (`/pmm-ui`) alongside ADRE, Investigations, and RTA. Product design defines a three-column QAN workspace with an embedded AI aside (Figma: PMM Use Cases With AI).

The Grafana plugin remains feature-complete but uses Ant Design / Grafana UI, complicates AI integration (floating widget vs contextual panel), and diverges from native navigation patterns.

## Decision

- Build a **native QAN page** at `/pmm-ui/qan` using MUI and `@percona/percona-ui`.
- Reuse existing **qan-api2** REST endpoints (`/v1/qan/*`); no new QAN backend for v1.
- Ship behind **`nativeQanEnabled`** server setting (Technical Preview, default `false`).
- Follow the **Figma design lock** ([native-qan-design-lock.md](../native-qan-design-lock.md)): three-zone layout (listing + Query Fingerprint section tabs + 400px AI aside), filters drawer + chips, advisory-only AI. Grafana QAN remains fallback while the flag is off — not a pixel port target.
- Embed a **400px AI aside** (ADRE chat + QAN context); hide global chat widget on QAN routes.
- **Advisory-only AI:** PMM recommends optimizations; customers copy and execute manually. No IDE integration, no auto-apply of DDL or query changes.

## Consequences

- Large UI port from `dashboards/pmm-app/src/pmm-qan/` to `ui/apps/pmm/src/pages/qan/`.
- Grafana QAN remains reachable via direct URL during transition.
- ADRE chat gains optional `qan_context` payload and nav-only frontend tools for native QAN.
- `pmm_ui_focus_qan_query` navigates to native QAN when the flag is enabled.
- RTA (`/pmm-ui/rta/*`) unchanged; unified via existing QanHeader tabs.
