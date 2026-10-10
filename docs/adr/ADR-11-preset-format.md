# ADR-11: Parser preset format

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15574, PMM-15587

## Context

Each file source needs a parser that turns lines into a timestamp, a severity and a message. PMM must ship parsers for common formats, and let admins add their own. In the prototype:
- custom parsers were pasted verbatim into a config that pmm-agent rendered with server credentials in scope;
- built-ins could be edited, so fixing a built-in would overwrite user edits;
- built-ins had mixed ids.

## Options

1. Built-ins and custom presets both live only in the database, all editable.
2. Built-ins live as versioned files in pmm-managed, read-only. Custom presets are filelog operator chains, checked and stored in the database.

## Decision

Option 2.

**Built-ins:**
- They ship as YAML files embedded in pmm-managed, and are versioned with the code.
- At startup they are upserted into the `log_parser_presets` table with `built_in = true`. If an existing custom preset has the same name as a new built-in, it is renamed `<name>_custom`, and its sources keep pointing at it.
- Built-ins can be cloned, but not changed or removed.

**Custom presets:**
- They are filelog operator lists.
- Only known operator types are allowed. `EXPR(`, `${` and `env(` are rejected.
- Each one is validated with the bundled `otelcol validate`.
- A preset in use can't be deleted.

**Every preset:**
- It may declare a minimum pmm-agent version. Older agents don't receive sources that use it.
- A change is pushed only to the agents that use the preset.

## Consequences

- Fixing a built-in is a code change that reaches every install on upgrade.
- Custom presets can't run expressions or read the environment. Multi-line handling (`recombine`) stays available.
- Built-ins have unit tests with real samples of each format, run against the same stanza version as the collector.
