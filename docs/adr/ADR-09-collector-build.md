# ADR-09: Collector build

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15577

## Context

The prototype downloaded the prebuilt `otelcol-contrib` distribution, hundreds of components, from GitHub at build time without verifying a checksum. Every component shipped is attack surface and size, and many could read or write files or open listeners.

## Options

1. The prebuilt `otelcol-contrib`.
2. The prebuilt `otelcol` core distribution, which lacks filelog, journald and ClickHouse.
3. A custom build with the OpenTelemetry Collector Builder (OCB), from pinned versions.

## Decision

Option 3. One OCB manifest in the repository, every module at one pinned collector release, and only these components:
- receivers: `filelog`, `journald`, `otlp`;
- processors: `resource`, `memory_limiter`, `batch`, `transform`, `probabilistic_sampler`, `tail_sampling`;
- exporters: `otlphttp`, `clickhouse`;
- extensions: `file_storage`.

CI checks the built binary's component list against an expected list. The same `otelcol` binary ships in the pmm-client packages, at `/usr/local/percona/pmm/tools/otelcol`, and through them in the PMM Server image.

## Consequences

- A smaller binary and attack surface. Go module checksums verify the sources.
- Adding a component is a deliberate change to the manifest and the expected list.
- Collector upgrades are explicit version bumps, and run the schema contract test (ADR-04).
