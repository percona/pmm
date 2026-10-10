# ADR-08: File-read trust

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15572, PMM-15577

## Context

An Admin on PMM Server can add a log source with any path on any node. Packaged pmm-agent runs as root (its systemd unit has no `User=`), and child processes inherit that user. A collector running as root could read any file on the host and ship it to ClickHouse.

## Options

1. Run the collector as root, and trust the server's choice of paths.
2. Run it as root, and check paths against an allow-list on the node.
3. Run it as the non-root `pmm-agent` user, **and** check paths against an allow-list on the node.

## Decision

Option 3.

**How the collector runs:**
- When pmm-agent runs as root, it starts the collector process alone with the uid, gid and supplementary groups of the `pmm-agent` user.
- If that user is missing, the collector doesn't start, and its status says why.
- When pmm-agent itself is not root, as in the Docker image, the collector runs as pmm-agent's user.
- Other agent types are unchanged.

**What the collector may read:**
- `pmm-agent.yaml` holds the allow-list:
  - `log-sources.allowed-paths`, default `["/var/log"]`;
  - `log-sources.allowed-journald-units`, default empty.
- The server can't change the allow-list.
- Symlinks are resolved before the check, and paths containing `..` are rejected.
- Database log paths that pmm-agent discovered itself, from the database (ADR-19), are accepted outside the allow-list. Paths the server merely labels as discovered are not.

**Packages:** they add `pmm-agent` to `systemd-journal`, and to `adm` on Debian/Ubuntu.

## Consequences

- The collector reads only what the `pmm-agent` user can read, inside the allow-list.
- On RHEL, MySQL, PostgreSQL and MongoDB logs are not readable by default. Users add an ACL or a group, and the source's status says so.
- Membership of `systemd-journal` lets the collector read every unit's journal. That is why hand-typed units need the allow-list (PMM-15591).
- The OCB build contains no component that writes files, other than the collector's own state store (ADR-09).
