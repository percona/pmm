# MCP endpoint for AI assistants

PMM Server exposes a [Model Context Protocol](https://modelcontextprotocol.io) (MCP) endpoint at `https://<pmm-server>/mcp`. Any MCP client, such as Claude Code, Cursor, or your own agent, can use it to triage slow queries with PMM data: list the monitored services, rank the worst queries from Query Analytics (QAN), fetch the metrics and an example statement for one query, get its execution plan and the DDL of its tables, and read the database server's configuration.

The endpoint is read-only. Every tool is registered with the MCP `readOnlyHint` and `destructiveHint: false` annotations, and no tool changes database or PMM state. MySQL EXPLAIN is always the non-executing form, never `EXPLAIN ANALYZE`. MongoDB explain runs the query's candidate plans to collect execution statistics. It costs about as much as the read itself, but it never modifies data.

## How it works

The MCP server runs inside pmm-managed and speaks MCP Streamable HTTP. It holds no credentials of its own: each tool call goes back out through PMM's public REST API with the caller's own `Authorization` header, so a client can never see or do more through MCP than it could with the same token and `curl`. Authorization is enforced per call by the same rules as for every other PMM API request, including [label-based access control](../admin/roles/access-control/intro.md) where it applies.

The endpoint is only reachable through PMM's nginx, which authenticates every request. A connection made directly to pmm-managed, bypassing nginx, is answered with 404 even when pmm-managed listens on a routable interface.

!!! note alert alert-primary "Service-account tokens and label-based access control"
    PMM resolves service-account tokens without a user ID, so label-based access control (LBAC) filters are not applied to them. A Viewer token sees the same services and queries a Viewer sees in the UI, subject to the role, but not to LBAC roles. This is existing PMM behaviour, not specific to MCP.

## Enable the endpoint

The endpoint is **off by default**: until it is enabled, `/mcp` answers 404. Enable it in either of these ways:

- Start PMM Server with the environment variable `PMM_ENABLE_MCP=true`, for example `docker run … -e PMM_ENABLE_MCP=true …`.
- Or change the setting on a running server, as an administrator. It takes effect immediately, without a restart:

    ```sh
    curl -X PUT https://<pmm-server>/v1/server/settings \
      -u admin:<password> \
      -H "Content-Type: application/json" \
      -d '{"enable_mcp": true}'
    ```

The current state is reported as `enable_mcp` in `GET /v1/server/settings/readonly`. Setting it back to `false` disables the endpoint again at once. When `PMM_ENABLE_MCP` is set, it takes precedence: a change that contradicts it is rejected.

## Create a token

The MCP client authenticates with a [service account token](authentication.md). Create one with the **Viewer** role: it is enough for every tool.
{.power-number}

1. Log in to PMM as an administrator.
2. From the side menu, click **Users and access > Service accounts**.
3. Click **Add service account**, give it a name such as `mcp-viewer`, select the **Viewer** role, and click **Create**.
4. Click **Add service account token**, give the token a name and, optionally, an expiry date, and click **Generate token**.
5. Copy the `glsa_…` value. It is shown only once.

A revoked or deleted token can keep working for up to 60 seconds, because pmm-managed caches the result of the authentication check.

## Connect a client

!!! warning "Use a client without unattended tools that send data out"
    Don't connect `/mcp` to an AI client that also runs unattended tools able to send data out of the client, such as web requests, email, chat or file upload. Text from your databases, such as query examples, table definitions and dimension values, can carry instructions. A client with such tools could be led to send PMM data elsewhere. Every MCP tool is read-only, so the risk is in the other tools that the client has.

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

The server sends the recommended order to the client on connect: `pmm_version`, `pmm_inventory`, `pmm_top_queries`, `pmm_query_detail`, then `pmm_get_explain` / `pmm_get_schema` and `pmm_get_config`. It also tells the client to pass the `query_ref` from `pmm_top_queries` or `pmm_query_detail` to `pmm_get_explain` and `pmm_get_schema`, to page with `offset`, and to read reference data from `pmm_catalog`. It tells the client to treat text from databases or users as data, not instructions: this includes QAN dimension values (user, client host, application name), `environment` and `cluster`. When a tool returns an error, the client must act on the remediation in the message and treat any database or user text that the message quotes as data.

| Tool | What it returns | Backing PMM API |
|------|-----------------|-----------------|
| `pmm_version` | PMM Server version; confirms the connection and the token | `GET /v1/server/version` |
| `pmm_inventory` | Monitored services with `service_id`, `service_name`, engine, version, node and address; optional `engine` / `node` filters; page with `limit` and `offset` | `GET /v1/inventory/services`, `GET /v1/inventory/nodes`, version metrics |
| `pmm_top_queries` | Rows of QAN's report over a window (`period_from` / `period_to`, RFC 3339 or relative such as `now-1h`): queries with their `queryid` by default, or another dimension with `group_by` (`service_name`, `database`, `schema`, `username`, `client_host`, `application_name`, `cmd_type`). Filter with `service_name`, `service_id`, `labels` and `search`; add up to 10 [QAN columns](#qan-columns); rank with `order_by` (`load`, `total_query_time`, `avg_query_time`, `count`, or a column); page with `offset` and `limit`. The header reads `Rows 11-20 of 143` | `POST /v1/qan/metrics:getReport`, `POST /v1/qan/metrics:getFilters` |
| `pmm_query_detail` | Fingerprint, engine and version, schema, tables, key metrics (rows examined/sent, full scans, filesorts, …; for MongoDB, documents and keys examined, documents returned and bytes read) and an example statement for one `queryid`. It reads only the services in your service list | `GET /v1/inventory/services`, `POST /v1/qan:getMetrics`, `POST /v1/qan/query:getExample`, version metrics |
| `pmm_get_explain` | The execution plan for a query, given as a `query_ref` or a `queryid` (see [Query references](#query-references)): the stored `pg_stat_monitor` plan for PostgreSQL, a live `EXPLAIN` (JSON or traditional) run by pmm-agent for MySQL, `explain` of the stored example for MongoDB | `GET /v1/inventory/services`, `POST /v1/qan/metrics:getReport`, `POST /v1/qan:getMetrics`, `POST /v1/qan/query:getExample`, `GET /v1/qan/query/{queryid}/plan` (PostgreSQL), `POST /v1/actions:startServiceAction`, `GET /v1/actions/{action_id}` and `POST /v1/actions:cancelAction` for an unfinished action (MySQL, MongoDB) |
| `pmm_get_schema` | `SHOW CREATE TABLE` (MySQL) or the table definition (PostgreSQL), the indexes, or both (`info_types`: `definition`, `indexes`, `all`) for 1 to 10 tables, fetched in parallel. Name the tables with `service_id`, `database` and `tables`, or give a `query_ref` or `queryid` to fetch the tables of that query (see [Query references](#query-references)). A table that fails shows its error in its own block; the call fails only when every table does | `GET /v1/inventory/services`, `POST /v1/actions:startServiceAction`, `GET /v1/actions/{action_id}`, `POST /v1/actions:cancelAction` for an unfinished action; with a `query_ref` or `queryid`, also `POST /v1/qan/metrics:getReport`, `POST /v1/qan:getMetrics` and `POST /v1/qan/query:getExample` |
| `pmm_catalog` | Reference data the other tools accept. `kind=qan_columns` lists the [QAN columns](#qan-columns) with their engine and aggregation; the optional `engine` argument keeps one engine's entries plus those every engine has | None: served from the server's built-in catalogue |
| `pmm_get_config` | The server's numeric configuration variables (`SHOW GLOBAL VARIABLES` style) plus the version, from the metrics PMM already collects; addressed by `service_id` like the other tools (`service_name` is also accepted); page with `limit` and `offset` | `GET /v1/inventory/services` (with `service_id`), `GET /graph/api/datasources/name/Metrics`, the Grafana datasource proxy |

Most results link to the same data in PMM, so you can check an answer in PMM itself. The link is the `pmm_url` field, and the text shows it too:

| Tool | The link opens |
|------|----------------|
| `pmm_top_queries` | QAN on the report's window, filtered to the service when the request names one with `service_name` |
| `pmm_query_detail` | QAN on the same window, with the query selected |
| `pmm_get_explain` | QAN on the window the query was read in, with the query selected on its **Explain** tab |
| `pmm_get_config` | QAN for the service, over the hour before `at` |
| `pmm_inventory` | The instance summary dashboard of each MySQL, PostgreSQL and MongoDB service |

`pmm_version`, `pmm_catalog` and `pmm_get_schema` carry no link. Links use the PMM public address setting when it is set, otherwise the host name the client connected to.

### Query references

`pmm_get_explain` and `pmm_get_schema` find a query by a query reference. A `query_ref` holds a `queryid` and the window the query was seen in (`period_from`, `period_to`). It can also hold the `service_id`. `pmm_top_queries` returns a `query_ref` on its `queryid` rows. `pmm_query_detail`, `pmm_get_explain` and `pmm_get_schema` return one as well.

Pass the saved `query_ref` back as it is. The tool then reads the same window, so you can explain a query that you saw yesterday. You can also pass a `queryid` alone, or beside `service_id`, `period_from` and `period_to`. Without `period_from` and `period_to`, a `queryid` searches the last hour, with or without `service_id`. A `query_ref` together with `queryid`, `service_id`, `period_from` or `period_to` is `invalid_input`.

The server resolves the query to one service in your service list, the same list that `pmm_inventory` returns. The call fails in these cases:

- The query was not seen in the window, on any of your services or on the `service_id` you gave: `not_found`, and the message names the window. Pass a window that covers when the query ran: `period_from` and `period_to`, or a `query_ref` whose period covers it.
- The query was seen on several of your services: `invalid_input`, and the message lists those services. Pass `service_id` to choose one. If you passed a `query_ref`, set the `service_id` inside the `query_ref` instead.
- The `service_id` is not in your service list: `not_found` with the text `not in your service list (services that sent metrics in the last hour)`. The message doesn't repeat the ID.

### QAN columns

`pmm_top_queries` adds up to 10 of these QAN metrics to each row (`columns`) and can rank by any of them (`order_by`, with `-` for descending). The meanings are QAN's own. A column shown as an average is per call; the others are totals over the window. The `order_by` column is shown as both, because QAN ranks a time metric by its per-call average and any other column by its total.

| Column | Engine | Shown as | Meaning |
|--------|--------|----------|---------|
| `load` | all | per second | Load |
| `num_queries` | all | sum | number of queries in bucket |
| `num_queries_with_errors` | all | sum | number of queries with errors |
| `num_queries_with_warnings` | all | sum | number of queries with warnings |
| `query_time` | all | average per call | Query Time |
| `lock_time` | all | average per call | Lock Time |
| `rows_sent` | all | average per call | Rows Sent |
| `rows_examined` | all | average per call | Rows Examined |
| `rows_affected` | all | average per call | Rows Affected |
| `rows_read` | all | average per call | Rows Read |
| `merge_passes` | MySQL | average per call | Merge Passes |
| `innodb_io_r_ops` | MySQL | average per call | Innodb IO R Ops |
| `innodb_io_r_bytes` | MySQL | average per call | Innodb IO R Bytes |
| `innodb_io_r_wait` | MySQL | average per call | Innodb IO R Wait |
| `innodb_rec_lock_wait` | MySQL | average per call | Innodb Rec Lock Wait |
| `innodb_queue_wait` | MySQL | average per call | Innodb Queue Wait |
| `innodb_pages_distinct` | MySQL | average per call | Innodb Pages Distinct |
| `query_length` | MySQL | average per call | Query Length |
| `bytes_sent` | MySQL | average per call | Bytes Sent |
| `tmp_tables` | MySQL | average per call | Tmp Tables |
| `tmp_disk_tables` | MySQL | average per call | Tmp Disk Tables |
| `tmp_table_sizes` | MySQL | average per call | Tmp Table Sizes |
| `qc_hit` | MySQL | sum | Query Cache Hit |
| `full_scan` | MySQL | sum | Full Scan |
| `full_join` | MySQL | sum | Full Join |
| `tmp_table` | MySQL | sum | Tmp Table |
| `tmp_table_on_disk` | MySQL | sum | Tmp Table on Disk |
| `filesort` | MySQL | sum | Filesort |
| `filesort_on_disk` | MySQL | sum | Filesort on Disk |
| `select_full_range_join` | MySQL | sum | Select Full Range Join |
| `select_range` | MySQL | sum | Select Range |
| `select_range_check` | MySQL | sum | Select Range Check |
| `sort_range` | MySQL | sum | Sort Range |
| `sort_rows` | MySQL | sum | Sort Rows |
| `sort_scan` | MySQL | sum | Sort Scan |
| `no_index_used` | MySQL | sum | No Index Used |
| `no_good_index_used` | MySQL | sum | No Good Index Used |
| `shared_blks_hit` | PostgreSQL | sum | Shared blocks cache hits |
| `shared_blks_read` | PostgreSQL | sum | Shared blocks read |
| `shared_blks_dirtied` | PostgreSQL | sum | Shared blocks dirtied |
| `shared_blks_written` | PostgreSQL | sum | Shared blocks written |
| `local_blks_hit` | PostgreSQL | sum | Local blocks cache hits |
| `local_blks_read` | PostgreSQL | sum | Local blocks read |
| `local_blks_dirtied` | PostgreSQL | sum | Local blocks dirtied |
| `local_blks_written` | PostgreSQL | sum | Local blocks written |
| `temp_blks_read` | PostgreSQL | sum | Temp blocks read |
| `temp_blks_written` | PostgreSQL | sum | Temp blocks written |
| `shared_blk_read_time` | PostgreSQL | sum | Time the statement spent reading shared blocks |
| `shared_blk_write_time` | PostgreSQL | sum | Time the statement spent writing shared blocks |
| `local_blk_read_time` | PostgreSQL | sum | Time the statement spent reading local_blocks |
| `local_blk_write_time` | PostgreSQL | sum | Time the statement spent writing local_blocks |
| `cpu_user_time` | PostgreSQL | sum | Total time user spent in query |
| `cpu_sys_time` | PostgreSQL | sum | Total time system spent in query |
| `plans_calls` | PostgreSQL | sum | Total number of planned calls |
| `plan_time` | PostgreSQL | sum | Total plan time spent in query |
| `wal_records` | PostgreSQL | sum | Total number of WAL (Write-ahead logging) records |
| `wal_fpi` | PostgreSQL | sum | Total number of FPI (full page images) in WAL (Write-ahead logging) records |
| `wal_bytes` | PostgreSQL | sum | Total bytes of WAL (Write-ahead logging) records |
| `wal_buffers_full` | PostgreSQL | sum | Total number of times WAL buffers become full |
| `parallel_workers_to_launch` | PostgreSQL | sum | Total number of parallel workers to launch |
| `parallel_workers_launched` | PostgreSQL | sum | Total number of parallel workers launched |
| `docs_returned` | MongoDB | average per call | Docs Returned |
| `response_length` | MongoDB | average per call | Response Length |
| `docs_scanned` | MongoDB | average per call | Docs Scanned |
| `docs_examined` | MongoDB | average per call | Total number of documents scanned during query execution |
| `keys_examined` | MongoDB | average per call | Total number of index keys scanned during query execution |
| `storage_bytes_read` | MongoDB | average per call | Total number of bytes read from storage during a specific operation |
| `storage_time_reading_micros` | MongoDB | average per call | Indicates the time, spent reading data from storage during an operation |
| `locks_database_time_acquiring_micros_read_shared` | MongoDB | average per call | Indicates the time, spent acquiring a read lock at the database level during an operation |
| `locks_global_acquire_count_read_shared` | MongoDB | sum | Number of times a global read lock was acquired during query execution |
| `locks_global_acquire_count_write_shared` | MongoDB | sum | Number of times a global write lock was acquired during query execution |
| `locks_database_acquire_count_read_shared` | MongoDB | sum | Number of times a read lock was acquired at the database level during query execution |
| `locks_database_acquire_wait_count_read_shared` | MongoDB | sum | Number of times a read lock at the database level was requested but had to wait before being granted |
| `locks_collection_acquire_count_read_shared` | MongoDB | sum | Number of times a read lock was acquired on a specific collection during operations |

### Structured output

Every tool returns one typed result and renders it twice from the same value: as text in `content`, and as JSON in `structuredContent`, which follows the tool's `outputSchema` from `tools/list`. A client can read whichever it prefers, and both always agree. Field names are snake_case and times are RFC 3339.

Results carry provenance. `pmm_url` is the link to the same data in PMM, as listed under [Tools](#tools). `pmm_top_queries` and `pmm_query_detail` carry `window`, the time range of the QAN data they read (`period_from`, `period_to`). `pmm_get_config` reads one point in time and carries `at` instead. A tool that works on one query also carries `query_ref`, as described under [Query references](#query-references): `pmm_top_queries` on its `queryid` rows, `pmm_query_detail`, `pmm_get_explain`, and `pmm_get_schema` in its query forms. The `query_ref` holds the `queryid`, the `service_id` when known, and the window the query was read in. A MySQL or MongoDB plan from `pmm_get_explain`, and the MySQL tables of `pmm_get_schema`, come from a live call, so they carry a `note`: they show what the database would choose now, not during the window. A result made of several parts, such as the tables of `pmm_get_schema`, sets `partial` to `true` when at least one part failed and at least one succeeded. The failed part reports its own error, for example `definition_error` for a table. When every part fails, the call is an error.

### Paging

`pmm_inventory`, `pmm_get_config` and `pmm_top_queries` page with `limit` and `offset`:

| Tool | Default `limit` | Maximum `limit` | Counts |
|------|-----------------|-----------------|--------|
| `pmm_inventory` | 50 | 200 | services |
| `pmm_get_config` | 200 | 1000 | variables |
| `pmm_top_queries` | 10 | 100 | rows |

A list that doesn't fit ends with `showing a–b of n; pass offset=… for more`. Pass that `offset` to get the next page. Other limits apply to one item: an execution plan is cut at 64 KiB, and the definition and indexes of one table at 32 KiB each. A cut item sets `truncated` and says so in its text.

### Untrusted fields

Some text in a result comes from your databases or from users: fingerprints, query examples, execution plans, table definitions, label and dimension values, variable values, the names of services, nodes, databases, schemas and tables, service addresses, version strings, and the message of a failed part's error. The `outputSchema` description of each such field starts with `Untrusted.`. Some clients, Claude Code among them, don't pass output schemas to the model, so the server's instructions also name these kinds of text and tell the model to treat them as data, not instructions. The instructions name more than the schemas mark: the QAN dimension values user, client host and application name, and the `environment` and `cluster` of a service.

This mark is advice to the model, not a security boundary. A model can still be misled by hostile text in a query. What bounds the damage is that every tool is read-only and runs with the caller's own token and roles.

### Errors

A failed tool call returns `isError: true` with a text of the form `error: <code>` followed by a remediation message. Codes: `not_found`, `invalid_input`, `unauthorized`, `timeout`, `no_query_source`, `insufficient_privileges`, `agent_unreachable`, `pmm_unavailable`, `rate_limited`. An `unauthorized` error carries PMM's own message (for example `Access denied` when the token's role is too low for a backing API); a request without a valid token never reaches the tools and gets HTTP 401 from PMM. When Grafana's anonymous access is on, PMM refuses `/mcp` to an anonymous caller with HTTP 403. A browser request from another origin also gets HTTP 403, so browser-based MCP clients must run on the PMM origin.

Every error result also carries `_meta["com.percona/error"]` with the `code` and a `retryable` flag, and has no `structuredContent`. A client can act on the flag without parsing the text. These codes are retryable, because the same call can succeed later: `timeout`, `agent_unreachable`, `pmm_unavailable` and `rate_limited`. The others (`not_found`, `invalid_input`, `unauthorized`, `no_query_source`, `insufficient_privileges`) are not: change the request or the setup first.

`rate_limited`: too many live-inspection calls are in flight for these credentials (2), on this replica (8), or on one service on this replica (1). The message doesn't name a service. Retry when one finishes.

Input that doesn't match a tool's input schema, such as a missing required argument or a value of the wrong type, is refused by the MCP SDK before the tool runs. That refusal has `isError: true` and a text that starts with `validating "arguments":`. It has no `error: <code>` text and no `com.percona/error` entry in `_meta`, so it is outside the error envelope above. Tools check the values and combinations of their arguments themselves and answer those with `invalid_input`.

## Roles

| Tool | Minimum role |
|------|--------------|
| `pmm_version`, `pmm_catalog`, `pmm_top_queries`, `pmm_query_detail`, `pmm_get_explain`, `pmm_get_schema`, `pmm_get_config` | Viewer |
| `pmm_inventory` | Viewer (`GET /v1/inventory/services` and `GET /v1/inventory/nodes` are readable by any authenticated user; inventory changes still require Admin) |

A Viewer token can start the read-only pmm-agent actions behind `pmm_get_explain` and `pmm_get_schema`, exactly as it can through the REST API, so those tools expose execution plans and table definitions to any user who can view dashboards.

## Configuration

| Environment variable | Default | Effect |
|----------------------|---------|--------|
| `PMM_ENABLE_MCP` | `false` | Enables the endpoint. When disabled, `/mcp` answers 404. The state is shown as `enable_mcp` in `GET /v1/server/settings/readonly`, and, unless this variable is set, it can also be changed at runtime: see [Enable the endpoint](#enable-the-endpoint). |
| `PMM_MCP_RAW_SQL` | `false` | Allows tool output to include literal values from your data: query examples, the explained statement, and the literals in execution plans and agent errors. When off, statements are shown as normalized fingerprints, plan literals are replaced with `?`, and agent errors that can quote a statement are withheld. Removing the variable turns raw SQL off at the next start. |
| `PMM_MCP_ACTION_TIMEOUT` | `15s` | How long one `pmm_get_explain` or `pmm_get_schema` call may take, all of its pmm-agent actions together. Actions still running then are cancelled on pmm-agent and end with a `timeout` error. Each pmm-agent action also stops on its own after 10 seconds, with the same error. |

Set them on the PMM Server container like other `PMM_*` variables, for example `-e PMM_ENABLE_MCP=true`. They are persisted in PMM settings at start-up.

!!! note alert alert-primary "Data sensitivity"
    Query fingerprints are normally free of literal values. Example statements, execution plans and agent errors can contain literal values from your data. So can a fingerprint that PMM stored unnormalized, such as a PostgreSQL query that `pg_stat_monitor` truncated.

    With `PMM_MCP_RAW_SQL` off, the default, MCP keeps those values out of tool output:

    - Example statements are replaced by their fingerprints.
    - Literals in execution plans and MongoDB fingerprints are replaced with `?`.
    - A SQL fingerprint that PMM stored unnormalized is withheld. If an unfiltered `pmm_top_queries` covers both MySQL and PostgreSQL data, PMM looks up each query's engine with one more report; a fingerprint that contains quoted text is withheld when its query ran under both engines, or when that lookup fails. To see such a fingerprint, filter by `service_name`.
    - `group_by=username` and `group_by=client_host` return database user names and client addresses, as the QAN dashboard does for the same role. `PMM_MCP_RAW_SQL` does not hide them.
    - An agent error that can quote the statement is withheld, and its error code is kept.

    Masking keeps the plan's shape: access type, table and index names, rows, and MongoDB's execution counters. A plan that can't be parsed for masking is withheld. Table definitions from `pmm_get_schema` are returned as they are, so `CHECK` and `DEFAULT` constants and partition bounds are not masked. To keep literals from being collected at all, disable query examples at the source with `pmm-admin add … --disable-queryexamples`. This also stops `pmm_get_explain` from resolving `?` placeholders automatically for MySQL.

## Limitations

- **MySQL EXPLAIN needs a query example.** pmm-agent resolves the `?` placeholders of a fingerprint from the stored example. Without one (examples disabled, or short-lived connections whose statements leave `performance_schema.events_statements_history` before pmm-agent samples it), the fingerprint is explained as it is. MySQL then answers a syntax error (1064), and the tool returns `invalid_input`. The message asks you to enable query examples for the service, or to pass `placeholders`, one value per `?` in order. `pmm_get_explain` also asks for `database`. `pmm_get_schema` also offers to name the tables yourself with `service_id`, `database` and `tables`. If the number of `placeholders` doesn't match the statement, the tool returns `invalid_input` too. That message asks for one value per `?` in the fingerprint, in order, or for none when QAN stored a query example. `pmm_get_schema` again offers `service_id`, `database` and `tables`.
- **PostgreSQL stored plans are not yet scoped to a service.** `pmm_get_explain` sends the `service_id` with the plan request, but QAN does not yet filter stored plans by it or by label-based access control. When two PostgreSQL services record the same `queryid`, the plan returned can belong to the other service, including one the caller's access roles hide.
- **PostgreSQL plans are stored, not live.** PMM has no live EXPLAIN action for PostgreSQL. `pmm_get_explain` returns the plan captured by `pg_stat_monitor`, which requires `pg_stat_monitor.pgsm_enable_query_plan = on`. PMM's [PostgreSQL setup](../install-pmm/install-pmm-client/connect-database/postgresql.md) recommends keeping it off because plan capture splits a query's statistics over several records; enable it only where plan retrieval matters more than exact timing. With `pg_stat_statements`, or with plan capture off, the tool returns `not_found`.
- **MongoDB EXPLAIN uses the stored example document.** `pmm_get_explain` has no `query` input. It explains the example that QAN stored for the query in the window. When QAN has no example, the tool returns `no_query_source`. Pick a window in which QAN stored one, or enable query examples for the service.
- **Some queries can't be explained.** `pmm_get_explain` returns `invalid_input` for a query that QAN stored trimmed because it exceeded pmm-agent's `max-query-length`, and for a MongoDB write or admin operation: MongoDB explains only `find`, `aggregate`, `count` and `distinct`. For a trimmed query, raise `max-query-length` on the agent or pick another query.
- **MySQL tables from a query are a best effort.** With a `query_ref` or `queryid`, `pmm_get_schema` runs one live `EXPLAIN` and reads the tables from its plan. The plan is the one the database would choose now, not the one used in the window, and the result carries a `note` that says so. Names that the plan doesn't tie to a base table, such as derived tables and CTEs, are listed in `unresolved` and not fetched. Tables beyond the 10-table cap, in plan order, are listed in `not_fetched`. To fetch them, name them with `service_id`, `database` and `tables`. If the `EXPLAIN` fails, the call fails with its error and no table definition is fetched.
- **PostgreSQL tables from a query come from QAN's example, and their definitions need the `search_path`.** With a `query_ref` or `queryid`, `pmm_get_schema` fetches the tables that QAN recorded with the example. With `pg_stat_monitor`, QAN records them in `schema.table` form. With `pg_stat_statements`, it records the table names without their schema. When QAN recorded none, the tool returns `no_query_source`. pmm-agent reads only the definitions of tables that the monitoring user can see without a schema prefix, on its `search_path`. For a table outside the `search_path`, the definition answers `not_found`, but the table's indexes still return, and so do the other tables.
- **`pmm_get_schema` doesn't support MongoDB.** In either form it returns `invalid_input` for a MongoDB service.
- **The query tools see only your service list.** `pmm_query_detail`, `pmm_get_explain` and `pmm_get_schema` read only the services that `pmm_inventory` returns for your token. A `service_id` outside that list returns `not_found`. Without a `service_id`, `pmm_query_detail` sums only the services in the list and returns only their examples.
- **`pmm_get_config` is limited to numeric variables**, because exporters publish numeric values only. String-valued knobs such as `sql_mode` or `innodb_flush_method` are not available.
- **Versions come from metrics.** PMM's inventory does not carry the server version, so `pmm_inventory` reads `mysql_version_info`, `pg_static` and `mongodb_version_info`; a service without metrics reports `unknown`.
