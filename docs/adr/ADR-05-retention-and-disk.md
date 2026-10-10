# ADR-05: Retention, size cap and disk watermark

- Status: Proposed. This was left open on 2026-09-24; the option below was chosen on 2026-10-10.
- Date: 2026-10-10
- Tickets: PMM-15567, PMM-15590, PMM-15571, PMM-15594

## Context

- OTEL is on by default on every ClickHouse profile, including the 8 GB one (ADR-12), and logs share ClickHouse's disk with QAN.
- A changed retention must apply to data already stored.
- A size cap and a disk watermark must stop logs from starving QAN.
- QAN drops daily partitions in qan-api2, on every replica. That is harmless only because a time-based drop is idempotent; a cap or watermark drop is not.
- There is no disk-space logic for ClickHouse yet.

## Options

1. **`ALTER TABLE … MODIFY TTL`** for retention. Changing it either rewrites parts or leaves existing parts on their old TTL until they merge. A separate job is still needed for the cap and the watermark.
2. **A daily partition drop for retention, cap and watermark,** in one job.

## Decision

Option 2. A leader-only job in pmm-managed runs every 10 minutes:
1. It drops each `otel` partition older than its signal's retention: `logs_retention_days` for `logs`, and `traces_retention_days` for tables whose names start with `otel_traces`.
2. While `otel` uses more than `max_disk_gb`, or the ClickHouse volume is used above `disk_watermark_percent` (default 80), it drops the oldest remaining partition. It never drops today's partition.
3. If today's partition alone still breaks a limit, it pauses ingest: it stops `otel-collector` on every replica and raises an alert. Node collectors buffer in their disk queues.
4. It resumes ingest once both values are under 90 % of their limits.

QAN tables are never touched.

## Consequences

- One mechanism covers all three limits, and a retention change takes effect on the next pass, with no part rewrites.
- Retention granularity is one day.
- A flood can pause ingest until the next day or until an admin raises the cap. Clients keep the data in their queues meanwhile, up to 1 GiB each by default.
- The cap and watermark defaults come from the capacity test (PMM-15592).
