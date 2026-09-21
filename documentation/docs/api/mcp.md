# MCP endpoint for AI assistants

PMM Server exposes a [Model Context Protocol](https://modelcontextprotocol.io) (MCP) endpoint at `https://<pmm-server>/mcp`. Any MCP client, such as Claude Code, Cursor, or your own agent, can use it to triage slow queries with PMM data: list the monitored services, rank the worst queries from Query Analytics (QAN), fetch the metrics and an example statement for one query, get its execution plan and the DDL of its tables, and read the database server's configuration.

The endpoint is read-only. Every tool is registered with the MCP `readOnlyHint` and `destructiveHint: false` annotations, no tool changes database or PMM state, and EXPLAIN is always the non-executing form, never `EXPLAIN ANALYZE`.

## How it works

The MCP server runs inside pmm-managed and speaks MCP Streamable HTTP. It holds no credentials of its own: each tool call goes back out through PMM's public REST API with the caller's own `Authorization` header, so a client can never see or do more through MCP than it could with the same token and `curl`. Authorization is enforced per call by the same rules as for every other PMM API request, including [label-based access control](../admin/roles/access-control/intro.md) where it applies.

!!! note alert alert-primary "Service-account tokens and label-based access control"
    PMM resolves service-account tokens without a user ID, so label-based access control (LBAC) filters are not applied to them. A Viewer token sees the same services and queries a Viewer sees in the UI, subject to the role, but not to LBAC roles. This is existing PMM behaviour, not specific to MCP.

## Create a token

The MCP client authenticates with a [service account token](authentication.md). Create one with the **Viewer** role: it is enough for every tool.
{.power-number}

1. Log in to PMM as an administrator.
2. From the side menu, click **Users and access > Service accounts**.
3. Click **Add service account**, give it a name such as `mcp-viewer`, select the **Viewer** role, and click **Create**.
4. Click **Add service account token**, give the token a name and, optionally, an expiry date, and click **Generate token**.
5. Copy the `glsa_…` value. It is shown only once.

## Connect a client

Point the client at `https://<pmm-server>/mcp` and send the token in the `Authorization` header.

=== "Claude Code"

    ```sh
    claude mcp add --transport http pmm https://<pmm-server>/mcp \
      --header "Authorization: Bearer glsa_…"
    ```

=== "Cursor"

    Add the server to `~/.cursor/mcp.json` (or the project's `.cursor/mcp.json`):

    ```json
    {
      "mcpServers": {
        "pmm": {
          "url": "https://<pmm-server>/mcp",
          "headers": {
            "Authorization": "Bearer glsa_…"
          }
        }
      }
    }
    ```

=== "Any client / curl"

    The endpoint accepts JSON-RPC over HTTP POST. A session is not required: each request carries its own credentials.

    ```sh
    curl -X POST https://<pmm-server>/mcp \
      -H "Authorization: Bearer glsa_…" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json, text/event-stream" \
      -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pmm_inventory","arguments":{}}}'
    ```

    Responses are server-sent event streams (`event: message` / `data: {…}`). Basic authentication with `service_token:glsa_…` or with a PMM user and password works as well.

If PMM Server runs with a self-signed certificate, configure the client to trust it or terminate TLS in front of PMM: the MCP endpoint is served by the same nginx listener as the UI and the REST API.

## Tools

The server sends the recommended order to the client on connect: `pmm_version`, `pmm_inventory`, `pmm_top_queries`, `pmm_query_detail`, then `pmm_get_explain` / `pmm_get_schema` and `pmm_get_config`.

| Tool | What it returns | Backing PMM API |
|------|-----------------|-----------------|
| `pmm_version` | PMM Server version; confirms the connection and the token | `GET /v1/server/version` |
| `pmm_inventory` | Monitored services with `service_id`, `service_name`, engine, version, node and address; optional `engine` / `node` filters | `GET /v1/inventory/services`, `GET /v1/inventory/nodes`, version metrics |
| `pmm_top_queries` | The worst queries for a service over a window (`period_from` / `period_to`, RFC 3339 or relative such as `now-1h`), ranked by `load`, `total_query_time`, `avg_query_time` or `count`, with their `queryid` | `POST /v1/qan/metrics:getReport` |
| `pmm_query_detail` | Fingerprint, engine and version, schema, tables, key metrics (rows examined/sent, full scans, filesorts, …) and an example statement for one `queryid` | `POST /v1/qan:getMetrics`, `POST /v1/qan/query:getExample` |
| `pmm_get_explain` | The execution plan for a `queryid`: the stored `pg_stat_monitor` plan for PostgreSQL, a live `EXPLAIN` (JSON or traditional) run by pmm-agent for MySQL, `explain` for MongoDB | `GET /v1/qan/query/{queryid}/plan`, `POST /v1/actions:startServiceAction`, `GET /v1/actions/{action_id}` |
| `pmm_get_schema` | `SHOW CREATE TABLE` (MySQL) or the table definition (PostgreSQL) for a table, optionally with its indexes | `POST /v1/actions:startServiceAction`, `GET /v1/actions/{action_id}` |
| `pmm_get_config` | The server's numeric configuration variables (`SHOW GLOBAL VARIABLES` style) plus the version, from the metrics PMM already collects | `GET /graph/api/datasources`, the Grafana datasource proxy |

Every result carries a **View in PMM** link that opens the QAN dashboard on the same window, service and query, so an answer can always be checked in PMM itself. Links use the PMM public address setting when it is set, otherwise the host name the client connected to.

### Errors

A failed tool call returns `isError: true` with a text of the form `error: <code>` followed by a remediation message. Codes: `not_found`, `invalid_input`, `unauthorized`, `timeout`, `no_query_source`, `insufficient_privileges`, `agent_unreachable`, `pmm_unavailable`. An `unauthorized` error carries PMM's own message (for example `Access denied` when the token's role is too low for a backing API); a request without a valid token never reaches the tools and gets HTTP 401 from PMM.

## Roles

| Tool | Minimum role |
|------|--------------|
| `pmm_version`, `pmm_top_queries`, `pmm_query_detail`, `pmm_get_explain`, `pmm_get_schema`, `pmm_get_config` | Viewer |
| `pmm_inventory` | Viewer (`GET /v1/inventory/services` and `GET /v1/inventory/nodes` are readable by any authenticated user; inventory changes still require Admin) |

A Viewer token can start the read-only pmm-agent actions behind `pmm_get_explain` and `pmm_get_schema`, exactly as it can through the REST API, so those tools expose execution plans and table definitions to any user who can view dashboards.

## Configuration

| Environment variable | Default | Effect |
|----------------------|---------|--------|
| `PMM_ENABLE_MCP` | `true` | Enables the endpoint. When disabled, `/mcp` answers 404. The state is shown as `enable_mcp` in `GET /v1/server/settings/readonly`. |
| `PMM_MCP_RAW_SQL` | `true` | Allows tool output to include statements with literal values: the stored query example in `pmm_query_detail` and the explained statement in `pmm_get_explain`. Set to `false` to emit only normalized text (fingerprints and PMM's `explain_fingerprint`). |
| `PMM_MCP_ACTION_TIMEOUT` | `15s` | How long `pmm_get_explain` and `pmm_get_schema` wait for pmm-agent to finish an action before returning a `timeout` error. |

Set them on the PMM Server container like other `PMM_*` variables, for example `-e PMM_ENABLE_MCP=false`. They are persisted in PMM settings at start-up.

!!! note alert alert-primary "Data sensitivity"
    Query fingerprints have their literal values stripped and are safe to share broadly. Example statements and execution plans can contain literal values from your data. To keep them out of tool output, set `PMM_MCP_RAW_SQL=false`, or disable query examples at the source with `pmm-admin add … --disable-queryexamples` (this also stops `pmm_get_explain` from resolving `?` placeholders automatically for MySQL). Plan bodies can still embed literals (for example `attached_condition` in MySQL's JSON format).

## Limitations

- **MySQL EXPLAIN needs a query example.** pmm-agent resolves the `?` placeholders of a fingerprint from the stored example. Without one (examples disabled, or short-lived connections whose statements leave `performance_schema.events_statements_history` before pmm-agent samples it), MySQL answers a syntax error and the tool returns `invalid_input` with instructions to pass `placeholders` and `database` explicitly.
- **PostgreSQL plans are stored, not live.** PMM has no live EXPLAIN action for PostgreSQL. `pmm_get_explain` returns the plan captured by `pg_stat_monitor`, which requires `pg_stat_monitor.pgsm_enable_query_plan = on`. PMM's [PostgreSQL setup](../install-pmm/install-pmm-client/connect-database/postgresql.md) recommends keeping it off because plan capture splits a query's statistics over several records; enable it only where plan retrieval matters more than exact timing. With `pg_stat_statements`, or with plan capture off, the tool returns `not_found`.
- **MongoDB EXPLAIN uses the stored example document** as its input; pass `query` to explain a specific statement.
- **`pmm_get_config` is limited to numeric variables**, because exporters publish numeric values only. String-valued knobs such as `sql_mode` or `innodb_flush_method` are not available.
- **Versions come from metrics.** PMM's inventory does not carry the server version, so `pmm_inventory` reads `mysql_version_info`, `pg_static` and `mongodb_version_info`; a service without metrics reports `unknown`.
