# Autonomous Database Reliability Engineer (ADRE) / HolmesGPT Integration

ADRE integrates [HolmesGPT](https://holmesgpt.dev) with PMM to provide AI-assisted database reliability analysis, chat, and alert investigation.

## Prerequisites

- HolmesGPT running in a container (or elsewhere) and reachable from the PMM server
- Optional: [mcp-clickhouse](https://github.com/ClickHouse/mcp-clickhouse) for ClickHouse/otel.logs/QAN analysis

## Configuration

1. Enable ADRE in **PMM Settings** (Configuration → Settings → Advanced) or on the ADRE page (admin only)
2. Set the **HolmesGPT base URL** (e.g. `http://holmesgpt:8080`)
3. If HolmesGPT requires authentication, include credentials in the URL: `http://user:password@holmesgpt:5050` or `http://:api-key@holmesgpt:5050` (API key as password)

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

## Adding custom tools to HolmesGPT

HolmesGPT supports two ways to add your own tools:

### 1. Custom toolsets (YAML)

Define tools as shell commands in a `toolsets.yaml` file. Each tool has a `name`, `description`, and `command`; the LLM infers parameters from `{{ variable }}` placeholders. Use this for scripts, `curl` calls to APIs, or `kubectl`/CLI commands.

- **CLI:** `holmes ask "your question" --custom-toolsets=toolsets.yaml`; after editing run `holmes toolset refresh`.
- **Helm:** Configure under `holmes.customToolsets` in your values.

See [HolmesGPT Custom Toolsets](https://holmesgpt.dev/data-sources/custom-toolsets/).

### 2. MCP servers (recommended for new integrations)

Implement an [MCP](https://modelcontextprotocol.io/) server that exposes tools; HolmesGPT connects to it and discovers tools dynamically.

- **Transport:** Prefer `streamable-http`: your server exposes an HTTP endpoint (e.g. `http://your-mcp:8000/mcp/messages`); HolmesGPT calls it with `mode: streamable-http`.
- **Config:** Add the server under `mcp_servers` in `~/.holmes/config.yaml` or in Helm under `holmes.mcp_servers`, with `config.url`, `config.mode`, optional `config.headers`, and `llm_instructions` (when/how the LLM should use it).

Example (config file):

```yaml
mcp_servers:
  my_tools:
    description: "My custom PMM tools"
    config:
      url: "http://my-mcp-server:8000/mcp/messages"
      mode: streamable-http
    llm_instructions: "Use these tools for schema, EXPLAIN, and index inspection when investigating database issues."
```

If your MCP server runs inside or alongside PMM, ensure HolmesGPT can reach it (network, auth, and security as discussed earlier).

See [HolmesGPT MCP Servers](https://holmesgpt.dev/data-sources/remote-mcp-servers/).

## Grafana panel render and dashboard links

When Holmes (or a tool) renders a Grafana panel image via PMM’s render API and includes an “Open in Grafana” link in the same message, follow this contract so the UI shows one correct link per panel:

1. **Use the render tool’s `dashboard_url`.** When the render tool (e.g. calling PMM `GET /v1/grafana/render?format=json`) returns `image_url` and `dashboard_url`, the model must use that exact `dashboard_url` for any “Open in Grafana” (or “Open the … panel”) link in the same message as the panel image. Do not construct the dashboard link from other parameters or default time ranges; otherwise the link can have the wrong timeframe.

2. **Match panel to narrative.** The panel id (and dashboard) used for the render must match what the model describes (e.g. if the answer says “QPS graph”, the rendered panel must be the QPS panel, not a different one like “MySQL Connections”).

3. **Duplicate links are suppressed by PMM.** Duplicate “Open in Grafana” links in markdown are suppressed by the PMM UI when they refer to a panel that already has a render image in the message; the only link shown is the one under the image (with the correct timeframe). So one link per panel from the render tool response is enough.

## API

PMM proxies requests to HolmesGPT. Endpoints (all require authentication):

| Method | Path | Description |
|--------|------|-------------|
| GET | /v1/adre/settings | Get ADRE settings (viewer or admin) |
| POST | /v1/adre/settings | Update ADRE settings (admin only in UI) |
| GET | /v1/adre/models | List available models |
| POST | /v1/adre/chat | Chat (non-streaming or streaming if `stream: true` in body) |
| GET | /v1/adre/alerts | Firing alerts from Grafana Alertmanager (requires ADRE enabled) |
| POST | /v1/adre/investigate | Investigate alerts (supports streaming) |
