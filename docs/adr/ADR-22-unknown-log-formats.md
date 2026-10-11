# ADR-22: Logs in unknown formats

- Status: Proposed
- Date: 2026-10-11
- Tickets: PMM-15567, PMM-15573, PMM-15574 (new ticket to be filed)

## Context

A file source needs a parser preset (ADR-11). For a format no preset covers, PMM-15573 defaults to `raw`: the whole line is the message, the timestamp is the time the line was read, and there is no severity. Users asked whether PMM can work out a new format by itself instead of someone writing a parser each time.

A wrong parser is worse than none. It assigns wrong timestamps and severities, and nothing flags the error.

## Options

1. Manual only: `raw`, or a preset someone picks or writes.
2. Apply an automatically detected parser with no confirmation.
3. A safe generic default, plus suggestions that an admin confirms.
4. Have a language model draft a parser from sample lines.

## Decision

Option 3.

**The `auto` preset.** It is a built-in, and becomes the default preset for file sources instead of `raw`.
- It reads JSON lines with common time, level and message keys.
- It reads leading ISO-8601, syslog and `dd Mon yyyy` timestamps, and the first level word.
- When nothing matches, it keeps the line unchanged, as `raw` does.
- It does not join multi-line entries.

**Suggestions.**
- Every preset can carry a `detect` regex for the first line of an entry; tests keep it in step with the parser.
- For a source, PMM scores the presets against the last 200 lines it has stored, and offers those that cover at least 90 % of them.
- An admin applies one explicitly.
- Sampling uses stored lines only, so pmm-agent gains no new way to read files.

**Option 4 is deferred,** not rejected:
- Sample lines can contain secrets and personal data.
- Crafted lines could steer the model (PMM-15591).

If it is built, it must:
- be opt-in;
- use redacted samples and a model the admin configures;
- produce output that passes the same checks as any custom preset;
- never apply anything without an admin's approval.

## Consequences

- Most unknown logs get a usable timestamp and severity with no setup.
- Multi-line entries in unknown formats still arrive as separate rows until a fitting preset is applied.
- Parsers change only through an admin action, so a guess never corrupts stored data.
