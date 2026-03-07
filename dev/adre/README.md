# Autonomous Database Reliability Engineer (ADRE) / HolmesGPT Integration

ADRE integrates [HolmesGPT](https://holmesgpt.dev) with PMM to provide AI-assisted database reliability analysis, chat, and alert investigation.

## Prerequisites

- HolmesGPT running in a container (or elsewhere) and reachable from the PMM server
- Optional: [mcp-clickhouse](https://github.com/ClickHouse/mcp-clickhouse) for ClickHouse/otel.logs/QAN analysis

## Configuration

1. Enable ADRE in **PMM Settings** (Configuration → Settings → Advanced) or on the ADRE page (admin only)
2. Set the **HolmesGPT base URL** (e.g. `http://holmesgpt:8080`)

HolmesGPT and PMM must be able to communicate. If using Docker, ensure they share a network or that HolmesGPT is reachable from the PMM host.

## HolmesGPT Configuration

Configure HolmesGPT to use PMM data sources:

- **Prometheus**: `https://<pmm-host>/victoriametrics/` (with auth if required)
- **Alertmanager**: `https://<pmm-host>/prometheus/alerts` (or internal URL if same network)

## ClickHouse (Logs, QAN)

HolmesGPT has no built-in ClickHouse toolset. To enable log and QAN analysis:

1. Run [mcp-clickhouse](https://github.com/ClickHouse/mcp-clickhouse) in a container
2. Point it at PMM’s ClickHouse (host, port, user, password must be reachable from HolmesGPT)
3. Add it as an MCP server in HolmesGPT config (streamable-http transport)
   - Example: `url: "http://mcp-clickhouse:8000/mcp/messages"`, `mode: streamable-http`

PMM does not run or configure mcp-clickhouse; you manage it and HolmesGPT configuration yourself.

## API

PMM proxies requests to HolmesGPT. Endpoints (all require authentication):

| Method | Path | Description |
|--------|------|-------------|
| GET | /v1/adre/settings | Get ADRE settings (viewer or admin) |
| POST | /v1/adre/settings | Update ADRE settings (admin only in UI) |
| GET | /v1/adre/models | List available models |
| POST | /v1/adre/chat | Chat (non-streaming or streaming if `stream: true` in body) |
| GET | /v1/adre/alerts | Firing alerts from VMAlert/Alertmanager (requires ADRE enabled) |
| POST | /v1/adre/investigate | Investigate alerts (supports streaming) |
