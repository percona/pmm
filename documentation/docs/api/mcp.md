# MCP endpoint for AI assistants

PMM Server exposes a [Model Context Protocol](https://modelcontextprotocol.io) (MCP) endpoint at `https://<pmm-server>/mcp`. Any MCP client, such as Claude Code, Cursor, or your own agent, can use it to triage slow queries with PMM data: list the monitored services, rank the worst queries from Query Analytics (QAN), fetch the metrics and an example statement for one query, get its execution plan and the DDL of its tables, read the database server's configuration, and query PMM's metrics.

The endpoint is read-only. Every tool is registered with the MCP `readOnlyHint` and `destructiveHint: false` annotations, and no tool changes database or PMM state. MySQL EXPLAIN is always the non-executing form, never `EXPLAIN ANALYZE`. MongoDB explain runs the query's candidate plans to collect execution statistics. It costs about as much as the read itself, but it never modifies data.

## How it works

The MCP server runs inside pmm-managed and speaks MCP Streamable HTTP. It holds no credentials of its own: each tool call goes back out through PMM's public REST API with the caller's own credentials, so a client can never see or do more through MCP than it could with the same token and `curl`. Authorization is enforced per call by the same rules as for every other PMM API request, including [label-based access control](../admin/roles/access-control/intro.md) where it applies. A token goes to PMM's API as `Authorization: Bearer <token>`, without cookies, whether the client sent it as a Bearer value or as Basic `api_key:<token>` or `service_token:<token>`. Basic credentials of a user and a session cookie go as sent.

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

The server sends the recommended order to the client on connect: `pmm_version`, `pmm_inventory`, `pmm_top_queries`, `pmm_query_detail`, then `pmm_get_explain` / `pmm_get_schema` and `pmm_get_config`. To see what is wrong now, it tells the client to call `pmm_alerts`. For metrics it tells the client to start with `pmm_service_health`, to read `pmm_list_metrics` and `pmm_catalog` (with `kind=metrics` and the service's `engine`) before it writes a `pmm_query_metrics` query, to write MetricsQL, to aggregate first and filter by `service_name` or `service_id`, and to use an instant query for the current value and a range query for a trend. It also tells the client to pass the `query_ref` from `pmm_top_queries` or `pmm_query_detail` to `pmm_get_explain` and `pmm_get_schema`, to page with `offset`, and to read reference data from `pmm_catalog`. It tells the client to treat text from databases or users as data, not instructions: this includes QAN dimension values (user, client host, application name), `environment` and `cluster`, and the names, labels, summaries and descriptions of alerts, and the links in them. When a tool returns an error, the client must act on the remediation in the message and treat any database or user text that the message quotes as data.

| Tool | What it returns | Backing PMM API |
|------|-----------------|-----------------|
| `pmm_version` | PMM Server version; confirms the connection and the token | `GET /v1/server/version` |
| `pmm_inventory` | The services in your service list: those whose `up` series you can read and that sent metrics in the last hour, with `service_id`, `service_name`, service type (`engine`), version, node, environment, cluster and `last_seen`; optional `engine` / `node` filters; page with `limit` and `offset` | `GET /graph/api/datasources/name/Metrics`, the datasource proxy's `/api/v1/query` on `up`, version metrics |
| `pmm_top_queries` | Rows of QAN's report over a window of at most [30 days](#qan-window) (`period_from` / `period_to`, RFC 3339 or relative such as `now-1h`): queries with their `queryid` by default, or another dimension with `group_by` (`service_name`, `database`, `schema`, `username`, `client_host`, `application_name`, `cmd_type`). Filter with `service_name`, `service_id`, `labels` and `search`; add up to 10 [QAN columns](#qan-columns); rank with `order_by` (`load`, `total_query_time`, `avg_query_time`, `count`, or a column); page with `offset` and `limit`. The header reads `Rows 11-20 of 143` | `POST /v1/qan/metrics:getReport`, `POST /v1/qan/metrics:getFilters` |
| `pmm_query_detail` | Fingerprint, engine and version, schema, tables, key metrics (rows examined/sent, full scans, filesorts, …; for MongoDB, documents and keys examined, documents returned and bytes read) and an example statement for one `queryid`, over a window of at most [30 days](#qan-window). It reads only the services in your service list | The service list, `POST /v1/qan:getMetrics`, `POST /v1/qan/query:getExample`, version metrics |
| `pmm_get_explain` | The execution plan for a query, given as a `query_ref` or a `queryid` (see [Query references](#query-references)): the stored `pg_stat_monitor` plan for PostgreSQL, a live `EXPLAIN` (JSON or traditional) run by pmm-agent for MySQL, `explain` of the stored example for MongoDB | The service list, `POST /v1/qan/metrics:getReport`, `POST /v1/qan:getMetrics`, `POST /v1/qan/query:getExample`, `GET /v1/qan/query/{queryid}/plan` (PostgreSQL), `POST /v1/actions:startServiceAction`, `GET /v1/actions/{action_id}` and `POST /v1/actions:cancelAction` for an unfinished action (MySQL, MongoDB) |
| `pmm_get_schema` | `SHOW CREATE TABLE` (MySQL) or the table definition (PostgreSQL), the indexes, or both (`info_types`: `definition`, `indexes`, `all`) for 1 to 10 tables, fetched in parallel. Name the tables with `service_id`, `database` and `tables`, or give a `query_ref` or `queryid` to fetch the tables of that query (see [Query references](#query-references)). A table that fails shows its error in its own block; the call fails only when every table does | The service list, `POST /v1/actions:startServiceAction`, `GET /v1/actions/{action_id}`, `POST /v1/actions:cancelAction` for an unfinished action; with a `query_ref` or `queryid`, also `POST /v1/qan/metrics:getReport`, `POST /v1/qan:getMetrics` and `POST /v1/qan/query:getExample` |
| `pmm_catalog` | Reference data the other tools accept. `kind=qan_columns` lists the [QAN columns](#qan-columns) with their engine and aggregation; the optional `engine` argument keeps one engine's entries plus those every engine has. `kind=metrics` lists the [metrics catalogue](#metrics-catalogue); `engine` keeps one engine's metrics (`mysql`, `postgresql`, `mongodb` or `node`) | None: served from the server's built-in catalogue |
| `pmm_query_metrics` | The result of a MetricsQL query on PMM's metrics: instant at `time`, or over a range with `period_from`, `period_to` and `step`. Returns the top `limit` series by maximum, as raw points or summaries. See [Metrics tools](#metrics-tools) | `GET /graph/api/datasources/name/Metrics`, then the datasource proxy: `/api/v1/query`, `/api/v1/query_range` and `/api/v1/series` |
| `pmm_list_metrics` | What the metrics hold: metric names that contain `search` (`kind=names`, the default), label names (`labels`), the values of one label (`values`) or the label sets of series (`series`). Pages with `limit` and `offset` | `GET /graph/api/datasources/name/Metrics`, then the datasource proxy: `/api/v1/label/{name}/values`, `/api/v1/labels` and `/api/v1/series` |
| `pmm_service_health` | The triage signals of one service and its node over a window: for each, min, avg, max, last and the time of the newest sample, or `no_data`, or its own error. It reports what was measured and judges nothing | `GET /graph/api/datasources/name/Metrics`, the datasource proxy: the service list, `/api/v1/query_range`, version metrics |
| `pmm_get_config` | The server's numeric configuration variables (`SHOW GLOBAL VARIABLES` style) plus the version, from the metrics PMM already collects; addressed by `service_id` like the other tools (`service_name` is also accepted); page with `limit` and `offset` | `GET /graph/api/datasources/name/Metrics`, the Grafana datasource proxy: the service list and the variables |
| `pmm_alerts` | The alerts firing now, most severe and newest first, with severity, labels, summary, description, start time, whether a silence mutes the alert and until when, and a link to its rule; optional `service_id`, `service_name`, `node_name` and `severity` filters; page with `limit` and `offset`. See [Alerts](#alerts) | Grafana's rules API (`GET /graph/api/prometheus/grafana/api/v1/rules`), Grafana's Alertmanager API for silences, and the service list |

Most results link to the same data in PMM, so you can check an answer in PMM itself. The link is the `pmm_url` field, and the text shows it too:

| Tool | The link opens |
|------|----------------|
| `pmm_top_queries` | QAN on the report's window, filtered to the service when the request names one with `service_name` |
| `pmm_query_detail` | QAN on the same window, with the query selected |
| `pmm_get_explain` | QAN on the window the query was read in, with the query selected on its **Explain** tab |
| `pmm_get_config` | QAN for the service, over the hour before `at` |
| `pmm_inventory` | The instance summary dashboard of each MySQL, PostgreSQL, MongoDB, HAProxy and ProxySQL service (ProxySQL's dashboard selects by node, so a ProxySQL service without a node name has no link) |
| `pmm_service_health` | The instance summary dashboard of the service |
| `pmm_alerts` | The alert rule of each alert in PMM's alerting |

`pmm_version`, `pmm_catalog`, `pmm_get_schema`, `pmm_query_metrics` and `pmm_list_metrics` carry no link. Links use the PMM public address setting when it is set, otherwise the host name the client connected to.

### Query references

`pmm_get_explain` and `pmm_get_schema` find a query by a query reference. A `query_ref` holds a `queryid` and the window the query was seen in (`period_from`, `period_to`). It can also hold the `service_id`. `pmm_top_queries` returns a `query_ref` on its `queryid` rows. `pmm_query_detail`, `pmm_get_explain` and `pmm_get_schema` return one as well.

Pass the saved `query_ref` back as it is. The tool then reads the same window, so you can explain a query that you saw yesterday. You can also pass a `queryid` alone, or beside `service_id`, `period_from` and `period_to`. Without `period_from` and `period_to`, a `queryid` searches the last hour, with or without `service_id`. A `query_ref` together with `queryid`, `service_id`, `period_from` or `period_to` is `invalid_input`.

The server resolves the query to one service in your service list for the query's window (see [Limitations](#limitations)); `pmm_inventory` lists the last hour. The call fails in these cases:

- The query was not seen in the window, on any of your services or on the `service_id` you gave: `not_found`, and the message names the window. Pass a window that covers when the query ran: `period_from` and `period_to`, or a `query_ref` whose period covers it.
- The query was seen on several of your services: `invalid_input`, and the message lists those services. Pass `service_id` to choose one. If you passed a `query_ref`, set the `service_id` inside the `query_ref` instead.
- The `service_id` is not in your service list: `not_found` with the text `not in your service list (services that sent metrics in the window searched)`, then the window searched. The message doesn't repeat the ID.

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

`pmm_inventory`, `pmm_get_config`, `pmm_top_queries`, `pmm_list_metrics` and `pmm_alerts` page with `limit` and `offset`:

| Tool | Default `limit` | Maximum `limit` | Counts |
|------|-----------------|-----------------|--------|
| `pmm_inventory` | 50 | 200 | services |
| `pmm_get_config` | 200 | 1000 | variables |
| `pmm_top_queries` | 10 | 100 | rows |
| `pmm_list_metrics` | 100 | 1000 | names, labels, values or series |
| `pmm_alerts` | 50 | 200 | alerts |

A list that doesn't fit ends with `showing a–b of n; pass offset=… for more`. Pass that `offset` to get the next page. Other limits apply to one item: an execution plan is cut at 64 KiB, and the definition and indexes of one table at 32 KiB each. A cut item sets `truncated` and says so in its text.

### Untrusted fields

Some text in a result comes from your databases or from users: fingerprints, query examples, execution plans, table definitions, label and dimension values, variable values, the names of services, nodes, databases, schemas and tables, metric label values, service addresses, version strings, alert names, labels, summaries and descriptions, and the message of a failed part's error. The `outputSchema` description of each such field starts with `Untrusted.`. Some clients, Claude Code among them, don't pass output schemas to the model, so the server's instructions also name these kinds of text and tell the model to treat them as data, not instructions. The instructions name more than the schemas mark: the QAN dimension values user, client host and application name, the `environment` and `cluster` of a service, and the links in alert summaries and descriptions.

Metric label values, and the names, labels, summaries and descriptions of alerts, are returned as stored, not masked: they aren't SQL statements, and the same caller sees them in PMM (ADR-07). Alert text is written by whoever authors the alert rule or the monitored system, so a client must treat it as untrusted. This mark is advice to the model, not a security boundary. A model can still be misled by hostile text in a query. What bounds the damage is that every tool is read-only and runs with the caller's own token and roles.

### Errors

A failed tool call returns `isError: true` with a text of the form `error: <code>` followed by a remediation message. Codes: `not_found`, `invalid_input`, `unauthorized`, `timeout`, `no_query_source`, `insufficient_privileges`, `agent_unreachable`, `pmm_unavailable`, `rate_limited`. An `unauthorized` error carries PMM's own message (for example `Access denied` when the token's role is too low for a backing API); a request without a valid token never reaches the tools and gets HTTP 401 from PMM. When Grafana's anonymous access is on, PMM refuses `/mcp` to an anonymous caller with HTTP 403. A browser request from another origin also gets HTTP 403, so browser-based MCP clients must run on the PMM origin.

Every error result also carries `_meta["com.percona/error"]` with the `code` and a `retryable` flag, and has no `structuredContent`. A client can act on the flag without parsing the text. These codes are retryable, because the same call can succeed later: `timeout`, `agent_unreachable`, `pmm_unavailable` and `rate_limited`. The others (`not_found`, `invalid_input`, `unauthorized`, `no_query_source`, `insufficient_privileges`) are not: change the request or the setup first.

`rate_limited`: a call was refused by a limit on how many calls a caller or a replica takes. The limits are listed under [Rate limits](#rate-limits). A refusal for call rate carries `retry_after` in `_meta`. A refusal for calls in flight doesn't: retry when one finishes. The message doesn't name a service.

A model-written metrics query that breaks a budget is `invalid_input`, and the message says which budget and how to fit it. See [Metrics tools](#metrics-tools).

Input that doesn't match a tool's input schema, such as a missing required argument or a value of the wrong type, is refused by the MCP SDK before the tool runs. That refusal has `isError: true` and a text that starts with `validating "arguments":`. It has no `error: <code>` text and no `com.percona/error` entry in `_meta`, so it is outside the error envelope above. Tools check the values and combinations of their arguments themselves and answer those with `invalid_input`.

## Alerts

`pmm_alerts` lists the alerts that are firing now. It answers what is wrong at this moment, with the severity, labels, summary and description of each alert, when it started, and a link to its rule in PMM. It doesn't silence or acknowledge anything.

The list comes from Grafana's rules API, called with your credentials. Grafana's folder permissions apply, so a Viewer gets what PMM's alert page shows to a Viewer. The result carries a `note` that repeats what the list covers.

### Filters and paging

All arguments are optional:

| Argument | Keeps |
|----------|-------|
| `service_id` | Alerts whose `service_id` label is this service ID |
| `service_name` | Alerts whose `service_name` label is this service name |
| `node_name` | Alerts whose `node_name` label is this node name |
| `severity` | Alerts of this severity, such as `critical`, `error` or `warning`; case-insensitive |
| `limit` | At most this many alerts per page: default 50, maximum 200 |
| `offset` | Skips this many alerts, for the next page |

An alert that lacks a label never matches a filter on that label. The most severe alerts come first (`critical`, `error`, `warning`, other severities, then none), the newest first within a severity.

### What the list holds

MCP narrows the list to the services and nodes in your service list over the last 7 days, to match its other results. This isn't a restriction on what you can read in PMM: the alert page in PMM shows the alerts that Grafana's permissions allow. An alert is kept in these cases:

- It comes from one of PMM's own rules: the PMM Server component rules, and the HA rules on an HA deployment. It is shown whatever service or node it names.
- It has no `service_id`, `service_name`, `node_id` or `node_name` label.
- Its service labels name a service in your service list. If it has no service labels, its node labels name a node in your service list.

A `node_name` filter for a node outside your service list answers `not_found`, and so does a `service_id` or `service_name` filter for a service outside it. This includes `pmm-server`: PMM's own alerts and the down alerts of RDS and Azure services name `node_name="pmm-server"`, but `pmm-server` isn't in your service list unless it sent `up` samples that you can read.

### Firing alerts only

`pmm_alerts` shows currently firing alerts. It doesn't show resolved alerts. It also doesn't show alerts that Grafana keeps firing after they recover (`keep_firing_for`, shown as **Recovering** in Grafana). PMM's alert templates don't use `keep_firing_for`.

Silenced alerts are included. An alert that a silence mutes has `silenced` set to `true` and `silenced_until` set to the end of the silence. When several silences mute one alert, `silenced_until` is the latest end.

### Alerts and the service list

A service that has sent no `up` sample for an hour leaves `pmm_inventory`, but its alerts stay in `pmm_alerts` for 7 days, because the list for alerts looks back 7 days. After 7 days without samples, its alerts leave `pmm_alerts` too.

### Not in MCP

Advisor results aren't in MCP (ADR-18). Use the **Advisors** page in PMM for them. See [Advisors](../advisors/advisors.md).

Alert names, labels, summaries and descriptions are returned as written (ADR-07). See [Untrusted fields](#untrusted-fields).

## Metrics tools

`pmm_query_metrics`, `pmm_list_metrics` and `pmm_service_health` read PMM's metrics from VictoriaMetrics. Each call goes through the Grafana datasource proxy with your own `Authorization` header, so [label-based access control](../admin/roles/access-control/intro.md) applies to every read. A model writes the queries, so the server checks each one against budgets before it runs anything.

### Which tool to use

Use the tools in this order:
{.power-number}

1. `pmm_service_health` reads the triage signals of one service. It's the first call for "is this service healthy?".
2. `pmm_catalog` with `kind=metrics` and the service's `engine` lists that engine's part of the [metrics catalogue](#metrics-catalogue), each metric with a ready query. `engine=node` lists the node's metrics.
3. `pmm_list_metrics` finds metrics that the catalogue doesn't list, and the label values to filter by.
4. `pmm_query_metrics` runs a MetricsQL query. Aggregate first, for example `sum by (service_name) (rate(mysql_global_status_questions[5m]))` or `topk(10, …)`, and filter by `service_name` or `service_id`. Use an instant query (`time`, default now) for the current value and a range query (`period_from`, `period_to`) for a trend.

`pmm_query_metrics` takes `query`, `time` or `period_from`, `period_to` and `step`, and `limit`. The default range is the last hour. `step` is a whole number of seconds, written as `30s`, `5m` or `1h`. Without `step`, the server chooses the smallest one that keeps the range within 2,000 points. Each query runs with a timeout of 25 seconds.

### Metrics catalogue

`pmm_catalog` with `kind=metrics` lists 153 metrics: 53 for MySQL, 43 for PostgreSQL, 19 for MongoDB and 38 for the node. Each entry has the metric `name`, its `engine`, its `meaning`, its `unit` as Grafana names it, its `type` (`counter`, which only grows, so query it with `rate`, or `gauge`), the `labels` on its samples, and a `query` to run with `pmm_query_metrics` after you replace `<service>` or, for a node metric, `<node>` with a name. The catalogue lists the metrics that the PMM dashboards and the `pmm_service_health` signals use. CI checks it against the output of the exporters for MySQL 8.0, PostgreSQL 17 and Percona Server for MongoDB 7.0. A metric that only other versions expose isn't listed: find it with `pmm_list_metrics`.

### Service health signals

`pmm_service_health` takes a `service_id` from `pmm_inventory`, and optionally `period_from` and `period_to` (default the last hour). The window is just under 7 days at most, since the signals also read about 5 minutes before it and a call reads at most 7 days. A longer window is `invalid_input`, and the message names the longest window that fits. It runs one fixed query for each signal, taken from the PMM dashboards: 9 for a MySQL service, 7 for PostgreSQL and 9 for MongoDB, then 6 for its node (CPU, load, memory, disk read and write latency, and free space). It runs at most 4 queries at a time.

Each signal reports `min`, `avg`, `max`, `last` and `last_at`, the time of the newest sample, with its unit and the metric it reads. A signal that returns no series is `no_data`. A signal whose query fails carries its own `error`, and the result sets `partial`. If every query fails, the call fails with the first error. A value that is `null` is NaN or infinite, for example a ratio whose denominator is 0. If a query returns several series, the signal reports the one with the largest maximum, and the note says how many there were.

The tool doesn't judge: it has no healthy or unhealthy verdict and no stale state. `no_data` on every signal means that the service sent none of these metrics in the window although its `up` series is there, for example because its exporter could not be scraped or could not reach the database. `no_data` on a node signal can also mean that an Access Role that filters on service labels hides the node's series. A service that is not in your service list for the window, one that an Access Role hides included, is `not_found`, and a service that is not MySQL, PostgreSQL or MongoDB is `invalid_input`.

### Execution budget

The server refuses these expressions before it sends any request. Every refusal is `invalid_input` with a hint:

- **Size.** An expression is at most 4 KiB.
- **Selectors.** An expression has at most 10 selectors. Each `or` group inside braces, as in `{a="x" or b="y"}`, counts as one selector.
- **Metric name.** Every selector names a metric: `__name__` with `=`, or with `=~` and a regular expression that starts with at least 3 literal bytes, as in `__name__=~"mysql_global_status_.*"`.
- **Scope.** A selector that matches the name with a regular expression also needs a `service_id`, `service_name`, `node_id` or `node_name` matcher. The matcher is either `=`, or `=~` where every `|`-separated alternative is a literal. `db1|db2` and `db\.prod` are scopes. `.+` and `db.prod` aren't.
- **Time read.** An expression reads at most 7 days. The time read is the range of the query plus the largest lookbehind of any path from the query to a selector. An instant query has a range of 0.
- **`@`.** The `@` modifier is refused, since it reads outside the query's time range.
- **Templates.** A MetricsQL `WITH` template and the built-in templates `ttf`, `ru`, `range_median` and `alias` are refused, because they expand before the checks run.
- **Series per value.** Functions that create a series for each distinct value, or whose output no budget bounds, are refused: `count_values`, `count_values_over_time`, `histogram` and `histogram_over_time`. `aggr_over_time` is refused when it names one of them in a string argument.
- **Characters.** A non-ASCII character or a backslash outside a quoted string is refused. Label values and other strings can hold both.
- **Subquery step.** The step of a subquery, written as `[range:step]`, is at least the time read divided by 2,000. A subquery without a step takes the step of the query and is checked the same way.
- **Instant range.** An instant `pmm_query_metrics` query can't be a range selector or a subquery, such as `x[5m]`, `x[1h:1m]` or `x[:1m]`, with or without `offset`, since VictoriaMetrics answers it with a range of samples. Wrap it in a rollup such as `last_over_time()` or `rate()`, or send a range query.

The lookbehind of a path is the sum of what the path passes through:

- the window of each rollup, such as `[5m]` in `rate(x[5m])`;
- the range of each subquery, plus two of its own steps and 5 minutes, which is where VictoriaMetrics starts it;
- each `offset`, by its absolute value.

A selector that is read through no window, such as `up`, or as the argument of a rollup function written without one, such as `rate(x)`, reads one step back. For an instant query the step is 5 minutes. A window shorter than the step counts as one step. For example, `rate(x[5m])` over the last 7 days reads 7 days and 5 minutes and is refused, while the last 6 days is accepted. `rate(x[6d])` as an instant query reads 6 days. `max_over_time(rate(x[6d])[6d:1h])` reads more than 12 days and is refused.

A rollup over an expression that isn't a selector, such as `rate(sum(x)[1h])`, runs in VictoriaMetrics as a subquery at the step of the query. It is limited to 4,000 points per series. If you didn't give `step`, the server raises it to what the rollup needs. If you did, a step that is too small is refused and the hint names the smallest one that fits.

### Series cap

Before a query runs, the server counts the series of each selector over the time read. A selector that matches more than 2,000 series refuses the query with `invalid_input`. Narrow it with a `service_*` or `node_*` matcher and aggregate.

### Response budget

A `pmm_query_metrics` result is shaped to fit these limits:

- **Series.** At most `limit` series, 20 by default and 100 at most. They are the series with the largest maximum.
- **Points.** At most 2,000 points in all.
- **Size.** At most 16 KiB, counting the text and the structured content together. When the query alone takes up that much, for example because JSON escapes each `<`, `>` and `&` to six bytes, the result shows only its first 200 bytes in `query`.
- **Summaries.** When the raw points don't fit in the points or in the size, every series carries a summary instead of points: `min`, `avg`, `max`, `last` and `last_at`. `summarized` is `true`.
- **Fewer series.** When even the summaries are over 16 KiB, the result keeps fewer series. `truncated` is `true`, and the hint says how many series it left out and how to narrow the query. `total_series` is the number the query returned.
- **Nulls.** NaN and infinite values are `null` in points and are left out of `min`, `avg` and `max`. The `note` says how many.
- **Labels.** PMM's internal labels `agent_id`, `agent_type`, `node_id`, `machine_id`, `instance` and `job` are dropped. A label with the same value on every series shown appears once, in `shared_labels`.

`pmm_list_metrics` looks back 1 hour by default and at most 7 days (`period_from`). MCP asks VictoriaMetrics for at most 1,001 entries, so `page.total` is at most 1,001. When VictoriaMetrics has more than 1,000, the result sets `truncated` and a hint to narrow the request. When `kind=names` finds more names than `limit`, the result lists groups of names by prefix with their counts instead. Series are also held to 16 KiB. `search` is a literal substring: a regular expression character in it, such as `.*`, matches only itself. The name search reads metric names only, so it doesn't need a metric name or a scope. The `match` selectors of the other kinds follow the [execution budget](#execution-budget) with a lookbehind of 0, and there are at most 10.

### Read cap

MCP reads at most 4 MiB of any response from the datasource proxy. A longer response fails the call. For a query or a selector that a model wrote, the error is `invalid_input` with a hint to narrow it with a `service_name` or `node_name` matcher and to aggregate. In other cases it's `pmm_unavailable`.

### Rate limits

The limits are per caller, identified by a hash of the credential, and per PMM Server replica:

| Limit | Applies to | Value |
|-------|------------|-------|
| Calls in flight, per caller | `pmm_query_metrics`, `pmm_list_metrics`, `pmm_service_health`, `pmm_top_queries`, `pmm_query_detail` | 4 |
| Calls in flight, per replica | The same tools | 8 |
| Call rate, per caller | `pmm_query_metrics`, `pmm_list_metrics`, `pmm_service_health` | 60 a minute, 10 at once |
| Live-inspection calls in flight, per caller | `pmm_get_explain` for MySQL and MongoDB, `pmm_get_schema` | 2 |
| Live-inspection calls in flight, per replica | The same calls | 8 |
| Live-inspection calls in flight, per service and replica | The same calls | 1 |
| Live-inspection call rate, per caller | The same calls | 10 a minute, 4 at once |

A call of `pmm_query_metrics`, `pmm_list_metrics` or `pmm_service_health` counts once, however many queries it runs. In addition, every request that MCP sends to VictoriaMetrics takes one of the replica's slots. The replica has half of VictoriaMetrics' limit on concurrent queries, at least 1, so dashboards and alert evaluation keep the other half. VictoriaMetrics' own default for that limit is twice the number of CPUs, at most 16. The environment variable `VM_search_maxConcurrentRequests` sets the limit instead (see [VictoriaMetrics environment variables](../reference/third-party/victoria.md#environment-variables)), and MCP takes half of the value it sets. A request waits for a slot until the deadline of its call.

MCP has no fleet size of its own. PMM's [hardware sizing](../install-pmm/plan-pmm-installation/hardware_and_system.md) applies. This is the setup that MCP was load-tested on, not a limit: a server with 4 CPU cores and 8 GB of memory that monitored 53 MySQL services and held about 2 days of data. Five clients called the metric tools at the caps for 10 minutes. No alert-rule evaluation failed and no PMM Server component restarted. The p95 of a dashboard query went from 47 ms to 170 ms, and VictoriaMetrics' CPU peaked at 2.3 cores.

A call refused for its rate is `rate_limited` and says in the message how many seconds to wait. `_meta["com.percona/error"]` carries the wait as `retry_after`, in whole seconds, at least 1. A call refused for the calls in flight is `rate_limited` without `retry_after`.

### QAN window

`pmm_top_queries` and `pmm_query_detail` read at most 30 days: a longer window is `invalid_input`, and the message asks you to narrow `period_from` or `period_to`. The two tools also take a slot of the calls-in-flight limits listed under [Rate limits](#rate-limits).

## Roles

| Tool | Minimum role |
|------|--------------|
| `pmm_version`, `pmm_inventory`, `pmm_catalog`, `pmm_top_queries`, `pmm_query_detail`, `pmm_get_explain`, `pmm_get_schema`, `pmm_get_config`, `pmm_query_metrics`, `pmm_list_metrics`, `pmm_service_health`, `pmm_alerts` | Viewer |

`pmm_inventory` reads no inventory API, only the metrics your token can read. `pmm_alerts` shows the alerts that Grafana's folder permissions let your token read.

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
- **PostgreSQL plans are stored, not live.** PMM has no live EXPLAIN action for PostgreSQL. `pmm_get_explain` returns the plan captured by `pg_stat_monitor`, which requires `pg_stat_monitor.pgsm_enable_query_plan = on`. PMM's [PostgreSQL setup](../install-pmm/install-pmm-client/connect-database/postgresql.md) recommends keeping it off because plan capture splits a query's statistics over several records; enable it only where plan retrieval matters more than exact timing. With `pg_stat_statements`, or with plan capture off, the tool returns `not_found`.
- **MongoDB EXPLAIN uses the stored example document.** `pmm_get_explain` has no `query` input. It explains the example that QAN stored for the query in the window. When QAN has no example, the tool returns `no_query_source`. Pick a window in which QAN stored one, or enable query examples for the service.
- **Some queries can't be explained.** `pmm_get_explain` returns `invalid_input` for a query that QAN stored trimmed because it exceeded pmm-agent's `max-query-length`, and for a MongoDB write or admin operation: MongoDB explains only `find`, `aggregate`, `count` and `distinct`. For a trimmed query, raise `max-query-length` on the agent or pick another query.
- **MySQL tables from a query are a best effort.** With a `query_ref` or `queryid`, `pmm_get_schema` runs one live `EXPLAIN` and reads the tables from its plan. The plan is the one the database would choose now, not the one used in the window, and the result carries a `note` that says so. Names that the plan doesn't tie to a base table, such as derived tables and CTEs, are listed in `unresolved` and not fetched. Tables beyond the 10-table cap, in plan order, are listed in `not_fetched`. To fetch them, name them with `service_id`, `database` and `tables`. If the `EXPLAIN` fails, the call fails with its error and no table definition is fetched.
- **PostgreSQL tables from a query come from QAN's example, and their definitions need the `search_path`.** With a `query_ref` or `queryid`, `pmm_get_schema` fetches the tables that QAN recorded with the example. With `pg_stat_monitor`, QAN records them in `schema.table` form. With `pg_stat_statements`, it records the table names without their schema. When QAN recorded none, the tool returns `no_query_source`. pmm-agent reads only the definitions of tables that the monitoring user can see without a schema prefix, on its `search_path`. For a table outside the `search_path`, the definition answers `not_found`, but the table's indexes still return, and so do the other tables.
- **`pmm_get_schema` doesn't support MongoDB.** In either form it returns `invalid_input` for a MongoDB service.
- **`pmm_query_detail`, `pmm_get_explain` and `pmm_get_schema` see only your service list.** The list is built for each call from the `up` series that your token can read, so [Access Roles](../admin/roles/access-control/intro.md) filter it. A service is in the list when it sent any `up` sample in the look-back, whatever the value: a target scraped with `up=0` stays listed, and `last_seen` is the time of its newest sample. The look-back is the call's window up to now, at most its last 7 days and at least 5 minutes, and the `not_found` says so when it cut the window. `pmm_inventory` and `pmm_get_schema` with named tables look back 1 hour, and `pmm_get_config` looks at the hour before `at`. `pmm_query_detail`, `pmm_get_explain` and `pmm_get_schema` read only the services in the list, and a `service_id` outside it returns `not_found`. Without a `service_id`, `pmm_query_detail` sums only the services in the list and returns only their examples. QAN itself can still show a service that the list lacks, for example one whose `up` series your Access Role hides: `pmm_top_queries`, which QAN filters by Access Role itself, can still rank it, but `pmm_query_detail`, `pmm_get_explain` and `pmm_get_schema` never return its data.
- **`pmm_get_config` is limited to numeric variables**, because exporters publish numeric values only. String-valued knobs such as `sql_mode` or `innodb_flush_method` are not available.
- **Versions come from metrics.** The service list carries no server version, so `pmm_inventory` reads `mysql_version_info`, `pg_static` and `mongodb_version_info`; a service without metrics, and one of another service type (ProxySQL, HAProxy, Valkey or external), reports `unknown`.
