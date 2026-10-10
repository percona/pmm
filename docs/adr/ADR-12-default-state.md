# ADR-12: Default state

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15571

## Context

A feature that is off by default is rarely tried, and its problems surface late. A feature that is on by default changes every install on upgrade, and shares ClickHouse with QAN.

## Options

1. Off by default; opt in through Settings.
2. On for new installs only.
3. On for new installs and after upgrade, on every ClickHouse profile.

## Decision

Option 3, labelled **Technical Preview**:
- `PMM_ENABLE_OTEL=false`, or the Helm value `otel.enabled=false`, keeps it off.
- Turning it on or off in Settings needs no restart.
- Turning it off stops the server collector and every node collector, and keeps the stored data. Purge deletes the data.

## Consequences

- The disk safeguards (ADR-05) and the separate ClickHouse users (ADR-04, ADR-07) are release blockers.
- The capacity test (PMM-15592), 50 nodes × 20 lines/s on the 8 GB profile, must pass before release.
- Upgrades start a new process and create a schema with no manual steps. Downgrade leaves the `otel` database and the new tables unused.
