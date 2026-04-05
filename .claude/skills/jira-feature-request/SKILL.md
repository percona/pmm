---
name: jira-feature-request
description: Draft a PMM Jira feature request ticket in the team's standard format. Use when filing a feature request, describing a new capability, or when the user mentions "feature request" or "Jira feature".
argument-hint: "[optional feature description or context]"
---

Draft a PMM Jira feature request ticket using the template below. All sections are mandatory — write "N/A" if a section genuinely does not apply. Use context from the current conversation, codebase, or `$ARGUMENTS` to fill in as much detail as possible.

## Title

Short, specific summary: `[Component] Brief description of the feature`

PMM components: QAN, Alerting, Backup, Inventory, UI, pmm-agent, pmm-managed, pmm-admin, vmproxy, Grafana, ClickHouse, VictoriaMetrics, Exporters, API.

## Description Fields

**User Story**
One or two sentences in the form "As a [persona], I want [capability] so that [benefit]." Name the persona specifically (e.g., DBA, DevOps engineer, PMM admin).

**Acceptance Criteria**
Bullet list of verifiable conditions that must be true for the feature to be considered done. Each criterion should be testable.

**Design / UI / UX (if applicable)**
Describe any UI changes, new screens, or UX flows. Attach mockups or wireframes if available. Write "N/A" for purely backend or API changes.

**Suggested Implementation**
High-level technical approach — relevant components, APIs, or data models involved. Not a full spec; just enough to scope the work. Write "N/A" if unknown.

**Out of Scope**
Explicitly list what this ticket does NOT cover to prevent scope creep.

**Details**
Any additional context: links to related tickets, customer reports, Slack threads, documentation references, or supporting data.

## Additional Fields

**How to document**
Brief note for the technical writer — new docs page, update to existing page, or release note entry? Reference the affected docs page if known.

**How to test**
Concrete verification steps for QA. Include: preconditions, test data setup, exact actions, and pass/fail criteria. Mention if it's automatable and which test suite it belongs to (api-tests, pmm-ui-tests, pmm-qa).

## Example

For the expected output format, see [example.md](example.md).
