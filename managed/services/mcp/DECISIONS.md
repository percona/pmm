# MCP server branch — decisions record

Decisions taken while implementing the MCP server, per
`pmm-mcp/docs/pmm-mcp-implementation-plan.md`. Each entry records the question,
the options, the answer and who decided. Answers D1–D6 were given by Ewen
Fortune (Sixta) on 2026-09-14 before Phase 0 started; D7–D9 are asked when
their phase is reached.

| # | Decision | Answer | Rationale |
|---|----------|--------|-----------|
| D1 | Where the branch lives | Fork `percona/pmm` to `sixta-systems/pmm`, branch `pmm-sixta` from `upstream/main`, draft PR to `percona/pmm:main` titled "[DRAFT — not for merge] MCP server for PMM" | Nobody outside Percona can push to `percona/pmm`; a draft PR makes the branch visible and reviewable there and Percona can pull it into an owned branch at will. |
| D2 | Commit/PR naming | Jira keys of the MCP epic PMM-14247 (phase 1 is PMM-15528) | Follows the repo's `PMM-XXXX` convention. |
| D3 | Minimum role for `GET /v1/inventory/services` and `GET /v1/inventory/nodes` | `viewer` (via `methodRules`) | A Viewer service account then covers the whole tool surface (least privilege). Viewers already see service and node names on every dashboard and in QAN filters; writes and `GET /v1/inventory/agents` stay admin. Fallback if Percona objects: `editor`. |
| D4 | Phase 4 scope (Access-Role permissions, service-account user IDs, UI) | Out of scope; §5.4 of the plan is shipped as a design proposal only: `dev/docs/managed/mcp-access-roles-proposal.md` | Keeps the branch small and reviewable; the RBAC direction is Percona's to set. |
| D5 | Default state of `PMM_ENABLE_MCP` | Off (`false`); surfaced in `GET /v1/server/settings/readonly` | PMM-15528: a new externally reachable, LLM-driven surface earns its default from field evidence and a formal security review (PMM-15532), not from a branch author's convenience. |
| D6 | Test/demo environment | Railway project (`pmm-mcp/demo/railway/`): `pmm-server` built from a multi-stage Dockerfile that compiles pmm-managed from this branch and layers it plus the nginx conf into `percona/pmm-server:3`; `percona-server` (MySQL 8); PostgreSQL with `pg_stat_monitor` and `pgsm_enable_query_plan=on`; `pmm-client` in push metrics mode; a workload loop | PMM Server has no arm64 build; Railway builds on x86_64, terminates TLS in front of nginx's plain 8080 listener, and is driven with the Railway CLI plus the PMM API instead of SSH. Fallback: an x86_64 VM. |
| D7 | Metrics path for `pmm_get_config` and the inventory version enrichment (asked at Phase 1, since enrichment needs it) | `GET /graph/api/datasources/proxy/uid/{uid}/api/v1/query`, uid looked up per query, with the caller's credentials, from `GET /graph/api/datasources/name/Metrics` (one small call, nothing cached across callers or left stale; a Viewer token may call it, live-checked 2026-10-02); `"/graph/api/datasources/proxy/uid/"` added to `lbacPrefixes` | Plain Prometheus response, Viewer-accessible (the raw `/prometheus` path is admin), live-verified on 3.8.1; the `lbacPrefixes` entry makes LBAC filters apply to it. Upstream `main` lists `/graph/api/datasources/uid` and the numeric proxy path, not the uid proxy path. |
| D8 | Streamable HTTP response mode | _asked in Phase 0 only if a client misbehaves behind nginx; SDK default (SSE) otherwise_ | |
| D9 | Redaction default (`PMM_MCP_RAW_SQL`) | `false`: statements shown as fingerprints, literals in plan bodies and agent errors masked with `?`; `true` opts in to literal values | PMM-15528: query examples and EXPLAIN bodies carry customer data, so sending them to a model is an opt-in. Masking (`redact.go`) keeps a plan's shape - access type, key, rows - so triage still works. It is fail-closed: every string in a plan, and every MongoDB number, is masked unless it is known to be plan structure, a statistic, or a key that holds only names (table, index, column), so a field a new engine version adds is masked too, and a plan that cannot be parsed is withheld. MySQL and PostgreSQL plan numbers are kept: they are estimates, and those engines print literals only inside expression strings. `pmm_get_schema` DDL is not masked. Agent errors and raw fingerprints are withheld rather than masked, since masking them relies on reading their quoting right: an error message is kept only when it is known to name only identifiers or hosts (pmm-agent's fixed texts, network errors, identifier-only MySQL numbers), and a SQL fingerprint with a literal outside its comments is a raw statement. When the engine of a report is unknown (several SQL engines in the window, or no answer from QAN), a quote that either MySQL or PostgreSQL reads as a string withholds the fingerprint. Raw SQL has no API setting, so a missing `PMM_MCP_RAW_SQL` means off at start-up. Operators needing a hard guarantee also disable query examples in PMM. |
| D10 | Dimension values in `pmm_top_queries` output (`group_by=username`, `client_host`) | Allowed with `PMM_MCP_RAW_SQL` off | Viewers already see them in QAN; D9 covers SQL literals, not user names or client addresses. Decided by Tibor Korocz on 2026-10-09 (PMM-15529). |

## Implementation notes that are decisions in their own right

- **`DisableLocalhostProtection: true`** on the Streamable HTTP handler, and a
  loopback-peer check in its place. nginx proxies to `127.0.0.1:7772` with the
  upstream name (`managed-json`) as the `Host` header, which the SDK's
  DNS-rebinding guard rejects as a non-loopback Host on a loopback connection -
  so the guard must be off, whatever interface pmm-managed is bound to. It would
  not protect against direct access either: the SDK only checks connections
  that arrive on a loopback address, and a direct one does not. What matters
  is that every request came through nginx `auth_request`, and nginx always
  connects from loopback, so `Handler` answers 404 to any non-loopback peer.
  That holds even when `PMM_INTERFACE_TO_BIND` makes port 7772 routable. The
  check covers `/mcp` only: the rest of the REST API on that port also relies
  on nginx alone, which is a server-wide question outside MCP.
  (PMM-15528 first tried keeping the SDK guard on for non-loopback binds; that
  turned every nginx-proxied request into a 403 and was reverted.)
- **`Stateless: true`**: no session table in pmm-managed, nothing to replicate
  in HA mode, restart-safe. GET/DELETE on `/mcp` answer 405 by SDK design.
- **Loopback override is `PMM_DEV_MCP_LOOPBACK_URL`**, not `PMM_MCP_LOOPBACK_URL`:
  the repo reserves the `PMM_DEV_*` prefix for development-only variables and
  `PMM_*` for GA functionality, and this override exists only for development.
- **Read-only inventory rules widen `GET /v1/inventory/services/{id}` and
  `GET /v1/inventory/nodes/{id}` too**, because `resolveRule` walks prefixes:
  a Viewer can read one service or node as well as the list. The fields are the
  same as in the list; writes and `GET /v1/inventory/agents` stay admin (tested).
- **`qan:getMetrics` and `qan/metrics:getReport` are decoded leniently, not with
  the generated go-swagger client.** Live on the demo (PMM 3.8.1-based image),
  qan-api2 answers sparkline fields with the strings `"NaN"` / `"Infinity"`
  (protojson's encoding of non-finite floats), which the swagger types reject
  (`cannot unmarshal string into ... float32`), so every `pmm_query_detail` call
  failed. The two calls now go through a raw POST into types whose floats accept
  numeric strings; sparklines are ignored. Worth a PMM-side fix (the Swagger spec
  declares `number`), listed under open items for Percona.
- **Branch name** `pmm-sixta` was chosen by the author (D1) rather than the
  repo's `PMM-XXXX-description` form; rename together with the PR when Percona
  issues a Jira key.
