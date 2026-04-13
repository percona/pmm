---
name: jira-feature-request
description: Draft a PMM Jira feature request in the team's standard format. Use when filing a feature request, proposing an enhancement, or when the user mentions "feature request" or "Jira feature".
argument-hint: "[optional feature description or context]"
---

Draft a PMM Jira feature request using the template below. All sections are mandatory — write "N/A" if a section genuinely does not apply. Use context from the current conversation, codebase, or `$ARGUMENTS` to fill in as much detail as possible.

## Title

Short, specific summary: `[Component] Brief description of the proposed feature`

PMM components: QAN, Alerting, Backup, Inventory, UI, pmm-agent, pmm-managed, pmm-admin, vmproxy, Grafana, ClickHouse, VictoriaMetrics, Exporters, API, DOC.

## Description Fields

**User Story**
One-sentence story in the canonical form:
`As a <role>, I want <capability>, so that <benefit>.`
Role should match a real PMM persona (DBA, SRE, developer, operator, platform admin). Capture the problem, not the solution.

**Acceptance criteria**
Bulleted, testable conditions that must all be true for the story to be considered done. Prefer Given/When/Then or plain observable outcomes. Each criterion should be verifiable by QA without access to the implementation — no "the code should...". Include negative cases and edge cases.

**Design / UI / UX (if applicable)**
Mockups, wireframes, Figma links, flow diagrams, or a textual description of the user flow. Call out new UI elements, where they live in the navigation, and behavior on loading/error/empty states. Write "N/A" for purely backend or API-only changes.

**Suggested implementation / options**
High-level technical approach(es) the author has in mind, with tradeoffs. List viable options (A/B/C) when there are real alternatives — e.g. "extend existing agent" vs. "new exporter" — and note pros/cons. Engineering owns the final decision; this section is input, not a spec.

**Out of scope**
Explicit list of things this ticket does NOT cover, to prevent scope creep. Reference follow-up tickets or future phases where appropriate (e.g. "MongoDB support — tracked in PMM-XXXX").

**Details**
Everything else needed to make the ticket actionable: links to related discussions (Slack, GitHub issues, support tickets), prior art from other monitoring tools, affected components, API references, dependencies on other teams or upstream projects, performance/scale considerations, security implications, and any relevant logs or data samples.

## Classification

Determine the request type:

- **New feature** — functionality that does not exist today in any form. Specify the target release if known.
- **Enhancement** — improvement to existing functionality (better UX, more options, performance, scalability). Reference the existing feature being enhanced.
- **Tech debt / Refactor** — internal-facing improvement with no direct user-visible change. Justify the cost in terms of future velocity, maintainability, or risk reduction.

State the classification explicitly in the ticket (e.g., "New feature — targeting 3.2.0" or "Enhancement to existing Backup scheduling").

## Example

For the expected output format, see [example.md](example.md).
