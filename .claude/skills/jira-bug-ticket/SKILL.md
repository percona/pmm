---
name: jira-bug-ticket
description: Draft a PMM Jira bug ticket in the team's standard format. Use when filing a bug report, describing a defect, or when the user mentions "bug ticket" or "Jira bug".
argument-hint: "[optional bug description or context]"
---

Draft a PMM Jira bug ticket using the template below. All sections are mandatory — write "N/A" if a section genuinely does not apply. Use context from the current conversation, codebase, or `$ARGUMENTS` to fill in as much detail as possible.

## Title

Short, specific summary: `[Component] Brief description of the defect`

PMM components: QAN, Alerting, Backup, Inventory, UI, pmm-agent, pmm-managed, pmm-admin, vmproxy, Grafana, ClickHouse, VictoriaMetrics, Exporters, API.

## Description Fields

**Steps to reproduce**
Numbered, minimal steps to reliably trigger the bug. Include PMM version, database type/version, and deployment method (Docker/OVF/AMI) when relevant.

**Actual result**
What happens — include error messages, HTTP status codes, or log snippets verbatim.

**Expected result**
What should happen according to documentation or intended behavior.

**User impact**
Who is affected (all users, specific DB type, specific deployment), severity (data loss, degraded monitoring, cosmetic), and whether it blocks a workflow.

**Workaround**
Temporary steps users can take to avoid the issue, or "None known."

**Details (+screenshots, whole logs)**
Attach screenshots, full log excerpts (not truncated) including PMM components logs, relevant config, and `pmm-admin status` / `pmm-admin list` output.

## Additional Fields

**How to document**
Brief note for the technical writer — does this need a Known Issues entry, a docs update, or a release note? Reference the affected docs page if known.

**How to test**
Concrete verification steps for QA. Include: preconditions, test data setup, exact actions, and pass/fail criteria. Mention if it's automatable and which test suite it belongs to (api-tests, pmm-ui-tests, pmm-qa).

## Classification

Determine the issue type:

- **Regression** — functionality that worked in a previous release but is now broken. Always fill in the `Affects version` field with the last known working version. Example: "Regression from 3.0.0 — worked in 2.x."
- **Defect** — new or changed functionality in the current release that was broken during development. `Affects version` is the current release only.

State the classification explicitly in the ticket (e.g., "Regression from 3.0.0" or "Defect introduced in 3.1.0").

## Example

For the expected output format, see [example.md](example.md).
