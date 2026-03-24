# ADR-001: PMM AI Investigations

## Status

Accepted.

## Context

PMM needs a first-class Investigations feature that combines:

- A configurable local LLM (Ollama by default) as the orchestrator for the user-facing chat.
- HolmesGPT as a tool the orchestrator can call for observability and database analysis.
- Persistent incident pages (reports) with blocks, comments, chat, and PDF export.
- Clear separation: normal chat is Q&A only; full investigation/report is triggered by an explicit "Run investigation" action and may involve a multi-turn loop between the orchestrator and HolmesGPT.

Existing ADRE (HolmesGPT) integration provides the HolmesGPT client and alerts; it does not provide persistent investigations, block-based reports, or orchestrator-driven routing.

## Decision

- **Orchestrator**: Stateless service that receives investigation context and chat messages, calls a configurable LLM (Ollama default) with a tool registry. The LLM decides when to call HolmesGPT vs other tools vs answer directly (routing via tool definitions and system prompt).
- **Investigations API**: REST API under `/v1/investigations` for CRUD on investigations, blocks, timeline, artifacts, comments, and messages. `POST /v1/investigations/:id/chat` invokes the orchestrator; `POST /v1/investigations/:id/run` (or equivalent) runs the full multi-turn investigation loop.
- **Data model**: New tables for investigations, investigation_blocks, investigation_artifacts, investigation_messages, investigation_comments, investigation_timeline_events. Blocks are ordered and typed (summary, timeline, single_panel, panel_group, logs_view, query_result, finding, markdown, etc.); content varies per incident.
- **No backward compatibility**: Replace ADRE direct-chat/investigate UX with Investigations; remove or make internal-only endpoints that are no longer needed.
- **Config**: Orchestrator LLM configurable via env vars (`PMM_ORCHESTRATOR_LLM_PROVIDER`, `PMM_ORCHESTRATOR_LLM_URL`, `PMM_ORCHESTRATOR_LLM_MODEL`) and PMM settings (stored in extended Adre or dedicated settings section).

## Consequences

- Single Incident Detail Page component; report content is data-driven (blocks from API).
- HolmesGPT is used as a tool; no change to HolmesGPT itself.
- Operators must run Ollama (or another configured LLM) for Investigations chat and "Run investigation" to work.

## Implementation note (tibi-holmes / current tree)

The shipped UI includes **both** **ADRE Chat** (floating widget) and **Investigations**; ADRE direct chat was not removed.

Investigation **chat** and **run** are implemented against the configured **HolmesGPT** URL (`adre.Client`) via **`POST /api/chat`**, with prompts and **`behavior_controls`** from PMM settings — not a separate in-repo Ollama orchestrator service. See `managed/services/investigations/chat.go` and [dev/investigations/README.md](https://github.com/percona/pmm/blob/v3/dev/investigations/README.md) for the actual request flow.

End-user overview: [AI features — Investigations](../use/ai-features/investigations.md).
