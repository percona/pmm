# Architecture decision records

This folder records significant architecture decisions for PMM, one decision per file.

**Format.** Each record has these sections:
- **Context:** why a decision was needed;
- **Options:** what was considered;
- **Decision:** what was chosen;
- **Consequences:** what follows from it, good and bad.

The header gives the status, the date, and the Jira tickets involved.

**Statuses:**
- **Proposed:** written and waiting for review by the PMM architects.
- **Accepted:** reviewed and in force.
- **Superseded by ADR-NN:** replaced; the record stays for history.

**Rules:**
- Never rewrite an accepted record to change its decision. Add a new ADR that supersedes it.
- Number records in sequence, as `ADR-NN-short-title.md`.

## Index

ADR-01 to ADR-14 record the OpenTelemetry decisions of the 24 Sep 2026 architecture review (epic [PMM-15567](https://perconadev.atlassian.net/browse/PMM-15567), ticket [PMM-15590](https://perconadev.atlassian.net/browse/PMM-15590)). ADR-15 and ADR-16 are reserved for eBPF ([PMM-15588](https://perconadev.atlassian.net/browse/PMM-15588)).

| ADR | Title | Status |
|---|---|---|
| [01](ADR-01-log-and-trace-store.md) | Log and trace store | Accepted |
| [02](ADR-02-identity-contract.md) | Identity contract | Accepted |
| [03](ADR-03-log-source-model.md) | Log-source model | Accepted |
| [04](ADR-04-schema-owner-and-layout.md) | Schema owner and table layout | Proposed |
| [05](ADR-05-retention-and-disk.md) | Retention, size cap and disk watermark | Proposed |
| [06](ADR-06-ingest.md) | Ingest path | Accepted |
| [07](ADR-07-log-access.md) | Who can read logs | Accepted |
| [08](ADR-08-file-read-trust.md) | File-read trust | Accepted |
| [09](ADR-09-collector-build.md) | Collector build | Accepted |
| [10](ADR-10-collector-placement.md) | Collector placement | Accepted |
| [11](ADR-11-preset-format.md) | Parser preset format | Accepted |
| [12](ADR-12-default-state.md) | Default state | Accepted |
| [13](ADR-13-alerting-on-logs.md) | Alerting on logs | Accepted |
| [14](ADR-14-otlp-export.md) | OTLP export | Accepted |
| 15, 16 | Reserved for eBPF (PMM-15588) | — |
| [17](ADR-17-profiles-deferred.md) | Profiles are deferred | Proposed |
| [18](ADR-18-otel-lbac.md) | LBAC for logs and traces | Proposed |
| [19](ADR-19-default-database-log-collection.md) | Database logs collected by default | Proposed |
| [20](ADR-20-collector-config-rendered-by-agent.md) | Node collector config rendered by pmm-agent | Proposed |
