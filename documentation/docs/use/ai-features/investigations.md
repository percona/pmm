# Investigations

**Investigations** are for a **deep dive** with a **structured report**: ordered **blocks** (findings, markdown, panels, query results, slow-query analysis, and more), **timeline** events, **comments**, chat messages, **PDF export**, and optionally a **ServiceNow** ticket.

In typical deployments on the **tibi-holmes** line, analysis is driven through **HolmesGPT** (PMM calls your Holmes deployment with investigation context). Your administrator ensures HolmesGPT is configured and reachable.

## When to use Investigations vs ADRE Chat

- Use [**ADRE Chat**](adre-chat.md) for quick, general Q&A.
- Use **Investigations** when you need a **persisted incident page**, multi-step analysis, **Run investigation**, and a shareable **PDF** (or ticket).

## What you can do in the UI

- **Create** an investigation (from an alert context or manually, depending on UI entry points in your build).
- **Open** the investigation detail page: view and reorder blocks, add comments, read the timeline.
- **Chat** inside the investigation (`POST` chat in API terms) for follow-up questions in context.
- **Run investigation** to execute the full analysis loop and populate or refresh blocks.
- **Export PDF** to download an HTML-based report.
- **ServiceNow** (if configured): create a ticket linked to the investigation when the UI offers that action.

## ServiceNow (optional)

Your admin configures:

- **ServiceNow URL** — endpoint your integration exposes for ticket creation (often a scripted REST or integration URL, not necessarily the interactive ServiceNow UI host).
- **API key** — sent as the `x-sn-apikey` header to that endpoint.
- **Client token** — application-specific token required by your integration payload.

Until all three are set in PMM **AI Assistant / ADRE** settings, ticket creation from Investigations will not be available. **Never** document or share real values for these settings.

## Privacy

Investigation content (titles, blocks, messages) is stored in **PMM’s database**. Analysis steps may send context to **HolmesGPT** according to your configuration. Apply the same data-handling rules as for ADRE Chat.

Technical API and flow details: [dev/investigations/README.md](https://github.com/percona/pmm/blob/v3/dev/investigations/README.md).

[← AI features overview](index.md)
