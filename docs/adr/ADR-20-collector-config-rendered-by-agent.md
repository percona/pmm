# ADR-20: Node collector config rendered by pmm-agent

- Status: Proposed. This changes the wording of PMM-15572 ("pmm-managed generates the collector config and pushes it").
- Date: 2026-10-10
- Tickets: PMM-15567, PMM-15572, PMM-15574

## Context

pmm-managed normally sends exporters their config as text files, which pmm-agent renders with Go `text/template`, with the server credentials in scope. For the collector, that config would contain user-supplied preset text and file paths. Two problems follow:
- Any template syntax in that text could expand the credentials into the config, and from there into stored records.
- pmm-agent can't apply its allow-list (ADR-08) to opaque YAML without parsing it.

## Options

1. pmm-managed renders the full YAML, with template delimiters chosen per config so they can't collide with user text.
2. pmm-managed sends typed parameters, and pmm-agent renders the YAML.

## Decision

Option 2.

**pmm-managed sends** an `OtelCollectorParams` message in `SetStateRequest`:
- the sources, with ids, paths or units, preset operator lists and identity attributes;
- the node identity;
- whether to send to the local receiver;
- the queue size;
- and later, the traces receiver.

**pmm-agent:**
- checks each source against its allow-list and for readability, and reports a state for each source;
- builds the collector YAML from typed structs, with no templates;
- fills in the server endpoint and credentials from its own config;
- escapes `$` in user strings as `$$`, so the collector's `${…}` expansion can't run;
- writes the config with mode 0600.

## Consequences

- No user text passes through a template, so this class of credential leak is gone.
- The allow-list is enforced per source, and each source reports why it is or isn't collecting.
- New collector features need a pmm-agent release, as new exporter flags already do. The server decides what to send by agent version.
