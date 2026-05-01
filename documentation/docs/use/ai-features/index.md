# AI features in PMM (ADRE, Investigations, QAN AI Insights)

PMM can connect to **[HolmesGPT](https://holmesgpt.dev)** so you can use AI-assisted analysis alongside metrics, Query Analytics (QAN), and dashboards. This section is written for **DBAs and SREs** who use the PMM UI.

For **operators** (URLs, APIs, Holmes configuration, Grafana panel rendering): see the developer docs linked at the end of this page.

## When to use which feature

| Feature | Use it when you want… |
| -------- | ---------------------- |
| **ADRE Chat** | **General chat** — quick questions, context-aware help, and conversation about your environment while you work in PMM. |
| **Investigations** | A **deep dive** with a structured **report** — blocks, timeline, running a full investigation, exporting **PDF**, and optionally creating a **ServiceNow** ticket if your admin configured it. |
| **QAN AI Insights** | **Query optimisation and tuning** — analysis focused on a specific query pattern in QAN, distinct from open-ended chat. |

## Glossary

| Term | Meaning |
| ---- | ------- |
| **ADRE** | Autonomous Database Reliability Engineer — PMM’s name for the AI assistant integration (including **ADRE Chat** in the UI). |
| **HolmesGPT** | The analysis backend PMM calls for many AI operations. Your organisation runs HolmesGPT where it can reach PMM APIs and (if configured) other data sources. |
| **Fast vs Investigation** | In **ADRE Chat**, **Fast** favours quick answers with lighter Holmes skills / TodoWrite usage by default; **Investigation** uses the full investigation-oriented controls. Admins tune Holmes **`behavior_controls`** and prompts under **AI Assistant** settings. |
| **Investigation** | A persisted incident page: title, status, **blocks** (findings, markdown, panels, query results, etc.), comments, and messages. |
| **Block** | A typed piece of content inside an investigation report (for example summary, finding, or slow-query analysis). |
| **QAN AI Insights** | AI-generated optimisation guidance for QAN data, with server-side caching per query and service. |

## Privacy and networking

- Messages and investigation runs are processed by the **configured backend** (typically **HolmesGPT**). Treat prompts and responses according to your organisation’s data policy.
- Use **TLS** and network policies so traffic between PMM and HolmesGPT is protected. **Do not** embed real passwords or API keys in URLs; store secrets in PMM settings or your secret manager as your admin defines.
- PMM may send **Grafana URL context** (path, variables, optional tab title) with ADRE Chat so the assistant knows which dashboard view you are on. That context is descriptive metadata, not a substitute for access control.

## Where to configure

- **Configuration → Settings** (Advanced / AI-related options) and the **AI Assistant / ADRE** area in the UI (exact labels depend on your PMM build).
- **ServiceNow** (optional): URL, API key, and client token are set in the same settings area when your organisation enables ticketing. Never share these values in chat or documentation.

## Further reading (technical)

Source files live in the PMM repository (not all are part of the published MkDocs tree):

- Holmes integration, APIs, Grafana render proxy, and operator notes: [dev/adre/README.md](https://github.com/percona/pmm/blob/v3/dev/adre/README.md)
- Investigations API and flows: [dev/investigations/README.md](https://github.com/percona/pmm/blob/v3/dev/investigations/README.md)
- Architecture decisions: [ADR-001 — PMM AI Investigations](../../adr/0001-pmm-ai-investigations.md), [ADR-002 — Data model and API](../../adr/0002-investigations-data-model-and-api.md)

- [ADRE Chat](adre-chat.md)
- [ADRE Slack bot](adre-slack-bot.md) (Socket Mode inside PMM Server)
- [Investigations](investigations.md)
- [QAN AI Insights](qan-ai-insights.md)
