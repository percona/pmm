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
claude mcp add --transport http pmm https://127.0.0.1/mcp \
  --header "Authorization: Bearer glsa_Fp0ggev31R58ueNJbJgYw7fIGfO3yKWH_746383ab"
```

Or call a tool directly with JSON-RPC:

```shell
curl -X POST --header 'Authorization: Bearer glsa_Fp0ggev31R58ueNJbJgYw7fIGfO3yKWH_746383ab' \
  --header 'Content-Type: application/json' --header 'Accept: application/json, text/event-stream' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pmm_inventory","arguments":{}}}' \
  https://127.0.0.1/mcp
```

The endpoint is controlled by the `PMM_ENABLE_MCP` environment variable (default `true`; `GET /v1/server/settings/readonly` reports it as `enable_mcp`). `PMM_MCP_RAW_SQL=false` restricts tool output to normalized statements without literal values. See the user documentation for the full tool reference, roles and limitations.
