# ADR-19: Database logs collected by default

- Status: Proposed. This changes PMM-15584, PMM-15585 and PMM-15586, which make collection opt-in.
- Date: 2026-10-10
- Tickets: PMM-15567, PMM-15573, PMM-15584, PMM-15585, PMM-15586

## Context

The epic adds `--collect-logs` to `pmm-admin add mysql|postgresql|mongodb`, and the tickets' tests pass it explicitly, so collection is opt-in. A flag users must know about leaves most installs without database logs. The other questions are where log paths come from, and which logs are safe to collect without asking:
- slow and general logs contain SQL text and personal data, and can be very large;
- QAN already reads the slow log.

## Options

**Default:**
1. opt-in;
2. the error log on by default;
3. all logs on by default.

**Paths:**
1. from the database only;
2. from the database, with distro default paths as a fallback.

## Decision

**What is collected by default:**
- `--collect-logs` defaults to `error` for MySQL, and is on for PostgreSQL and MongoDB, which have one log each.
- This applies only when the service is on the same host as pmm-agent: a socket, a loopback address, or one of the host's own addresses.
- `--collect-logs=none` (MySQL) or `--no-collect-logs` opts out.
- Slow and general logs stay opt-in.

**Where paths come from:**
- Paths come from the database only:
  - MySQL: `@@log_error`, resolved against `@@datadir`;
  - PostgreSQL: `log_directory`, `log_filename`, `data_directory` and `log_destination`;
  - MongoDB: `getCmdLineOpts`.
- pmm-agent discovers them, because it connects to the database and can see the files.
- Discovery runs again on every pmm-agent reconnect.

**When nothing can be collected:**
- The service is still added, and pmm-admin prints why: remote host, missing file, unreadable file, logs going to syslog.
- A new pmm-admin doesn't send the field to a PMM Server older than this feature.

## Consequences

- Most local database installs get error logs with no extra steps.
- No default path is guessed. Guessing would attribute the wrong file whenever two instances share a host, or the layout is custom.
- Remote and RDS services get no source.
- The epic owner must agree, because this changes the tickets' acceptance criteria.
