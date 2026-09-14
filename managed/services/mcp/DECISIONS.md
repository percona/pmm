# MCP server branch — decisions record

Decisions taken while implementing the MCP server, per
`pmm-mcp/docs/pmm-mcp-implementation-plan.md`. Each entry records the question,
the options, the answer and who decided. Answers D1–D6 were given by Ewen
Fortune (Sixta) on 2026-09-14 before Phase 0 started; D7–D9 are asked when
their phase is reached.

| # | Decision | Answer | Rationale |
|---|----------|--------|-----------|
| D1 | Where the branch lives | Fork `percona/pmm` to `sixta-systems/pmm`, branch `pmm-sixta` from `upstream/main`, draft PR to `percona/pmm:main` titled "[DRAFT — not for merge] MCP server for PMM" | Nobody outside Percona can push to `percona/pmm`; a draft PR makes the branch visible and reviewable there and Percona can pull it into an owned branch at will. |
| D2 | Commit/PR naming without a Jira key | `PMM-0000` placeholder prefix; PR body states the key is to be assigned by Percona | Keeps the repo's `PMM-XXXX` convention parseable; rename when a key arrives. |
| D3 | Minimum role for `GET /v1/inventory/services` and `GET /v1/inventory/nodes` | `viewer` (via `methodRules`) | A Viewer service account then covers the whole tool surface (least privilege). Viewers already see service and node names on every dashboard and in QAN filters; writes and `GET /v1/inventory/agents` stay admin. Fallback if Percona objects: `editor`. |
| D4 | Phase 4 scope (Access-Role permissions, service-account user IDs, UI) | Out of scope; §5.4 of the plan is shipped as a design proposal in `docs/` only | Keeps the branch small and reviewable; the RBAC direction is Percona's to set. |
| D5 | Default state of `PMM_ENABLE_MCP` | On in this branch; surfaced in `GET /v1/server/settings/readonly` | The endpoint is the thing under review; Percona decides the GA default. |
| D6 | Test/demo environment | Railway project (`pmm-mcp/demo/railway/`): `pmm-server` built from a multi-stage Dockerfile that compiles pmm-managed from this branch and layers it plus the nginx conf into `percona/pmm-server:3`; `percona-server` (MySQL 8); PostgreSQL with `pg_stat_monitor` and `pgsm_enable_query_plan=on`; `pmm-client` in push metrics mode; a workload loop | PMM Server has no arm64 build; Railway builds on x86_64, terminates TLS in front of nginx's plain 8080 listener, and is driven with the Railway CLI plus the PMM API instead of SSH. Fallback: an x86_64 VM. |
| D7 | Metrics path for `pmm_get_config` | _asked in Phase 2_ | |
| D8 | Streamable HTTP response mode | _asked in Phase 0 only if a client misbehaves behind nginx; SDK default (SSE) otherwise_ | |
| D9 | Redaction default (`PMM_MCP_RAW_SQL`) | _asked in Phase 2_ | |

## Implementation notes that are decisions in their own right

- **`DisableLocalhostProtection: true`** on the Streamable HTTP handler. nginx
  proxies to `127.0.0.1:7772` with the upstream name (`managed-json`) as the
  `Host` header, which the SDK's DNS-rebinding guard would reject as a
  non-loopback Host on a loopback listener. The guard exists for MCP servers
  bound to localhost without any other front door; here nginx `auth_request`
  is the front door and port 7772 is not reachable from outside the container.
- **`Stateless: true`**: no session table in pmm-managed, nothing to replicate
  in HA mode, restart-safe. GET/DELETE on `/mcp` answer 405 by SDK design.
- **Branch name** `pmm-sixta` was chosen by the author (D1) rather than the
  repo's `PMM-XXXX-description` form; rename together with the PR when Percona
  issues a Jira key.
