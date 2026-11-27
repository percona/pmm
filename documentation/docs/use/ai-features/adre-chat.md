# ADRE Chat

**ADRE Chat** is the floating chat in the PMM UI for **general conversation** with the AI assistant: questions about your environment, metrics, alerts, and how to interpret what you see.

It is different from [**Investigations**](investigations.md) (structured reports and deep runs) and [**QAN AI Insights**](qan-ai-insights.md) (query-focused tuning).

## Requirements

- An administrator must **enable ADRE** and set the **HolmesGPT base URL** in PMM **AI Assistant** settings.
- **HolmesGPT** must be reachable from the PMM server over your network. Use HTTPS in production where possible.

## What gets sent to the backend

- Your **messages** and a **short window** of recent chat history.
- A **system** preamble that includes PMM context. When you are on **Grafana** routes, PMM may attach **current Grafana context** derived from the URL synced with the Grafana iframe (dashboard UID, optional focused panel, time range, template variables, and sometimes the browser tab title). That helps the assistant answer “what am I looking at?” without guessing.

## Fast vs Investigation

The ADRE panel can run in **Fast** or **Investigation** mode. PMM sends Holmes **`behavior_controls`** and an **`additional_system_prompt`** appropriate to the mode (tunable under **Configuration → AI Assistant**). Operators should be aware that the Holmes container environment variable **`ENABLED_PROMPTS`** can override what the API may enable.

Technical details: [dev/adre/README.md](https://github.com/percona/pmm/blob/v3/dev/adre/README.md) and [Holmes fast mode / prompt controls](https://holmesgpt.dev/dev/reference/http-api/?h=fast#fast-mode--prompt-controls).

## Good practices

- Do not paste **passwords**, **connection strings with secrets**, or **API keys** into chat.
- If answers reference **panel images**, PMM may show rendered graphs; technical details of the image proxy are documented under **developer** docs only.

[← AI features overview](index.md)
