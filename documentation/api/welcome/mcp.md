---
title: MCP endpoint
slug: mcp
category:
  uri: welcome
position: 3
---

## MCP endpoint

PMM Server exposes a [Model Context Protocol](https://modelcontextprotocol.io) (MCP) endpoint at `/mcp` (Streamable HTTP) so that AI assistants such as Claude Code or Cursor can triage slow queries with PMM data. The endpoint is read-only and authenticates with the same [service account tokens](authentication.md) as the REST API; every tool call is executed through the REST API with the caller's token, so the token's role applies unchanged.

Tools: `pmm_version`, `pmm_inventory`, `pmm_top_queries`, `pmm_query_detail`, `pmm_get_explain`, `pmm_get_schema`, `pmm_get_config`. A Viewer token is sufficient for all of them.

Register the endpoint in Claude Code:

```shell
claude mcp add --transport http pmm https://<pmm-server>/mcp \
  --header "Authorization: Bearer <service-token>"
```

Or call a tool directly with JSON-RPC:

```shell
curl -X POST --header 'Authorization: Bearer <service-token>' \
  --header 'Content-Type: application/json' --header 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pmm_inventory","arguments":{}}}' \
  https://<pmm-server>/mcp
```

The endpoint is off by default. Enable it with the `PMM_ENABLE_MCP=true` environment variable, or at runtime with `PUT /v1/server/settings` and `{"enable_mcp": true}` unless that variable is set; `GET /v1/server/settings/readonly` reports the state as `enable_mcp`. Literal values from your data are masked in query fingerprints and examples, execution plans and agent errors unless `PMM_MCP_RAW_SQL=true` opts in to them; table definitions are returned as they are. See the user documentation for the full tool reference, roles and limitations.
