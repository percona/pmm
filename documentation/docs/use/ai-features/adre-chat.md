# ADRE Chat

**ADRE Chat** is the floating chat in the PMM UI for **general conversation** with the AI assistant: questions about your environment, metrics, alerts, and how to interpret what you see.

It is different from [**Investigations**](investigations.md) (structured reports and deep runs) and [**QAN AI Insights**](qan-ai-insights.md) (query-focused tuning).

## Requirements

- An administrator must **enable ADRE** and set the **HolmesGPT base URL** (or configure the **PMM Agent** chat backend) in PMM settings.
- **HolmesGPT** must be reachable from the PMM server over your network. Use HTTPS in production where possible.

## What gets sent to the backend

- Your **messages** and a **short window** of recent chat history.
- A **system** preamble that includes PMM context. When you are on **Grafana** routes, PMM may attach **current Grafana context** derived from the URL synced with the Grafana iframe (dashboard UID, optional focused panel, time range, template variables, and sometimes the browser tab title). That helps the assistant answer “what am I looking at?” without guessing.

## Backends (conceptual)

| Setting (conceptual) | Behaviour |
| -------------------- | --------- |
| **HolmesGPT** (`holmesgpt`) | PMM proxies chat to your HolmesGPT deployment. |
| **PMM Agent** (`holmes_agent`) | Chat uses the PMM Agent path with a configurable system prompt and history length. |

Exact field names are documented for operators in the PMM repo: [dev/adre/README.md](https://github.com/percona/pmm/blob/v3/dev/adre/README.md).

## Good practices

- Do not paste **passwords**, **connection strings with secrets**, or **API keys** into chat.
- If answers reference **panel images**, PMM may show rendered graphs; technical details of the image proxy are documented under **developer** docs only.

[← AI features overview](index.md)
