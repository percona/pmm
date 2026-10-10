# OTel 06: LBAC for Logs and Traces (design level, re-plan before execution)

> **For agentic workers:** this is a **design-level** plan for a follow-up epic that doesn't exist yet. Do not execute it as written. Before starting:
> 1. Get the ADR in Task 0 accepted.
> 2. Re-run superpowers:writing-plans against the merged code from plans 01–05.

**Goal:** Users restricted by LBAC see the logs and traces of exactly the services and nodes their roles allow, with the same semantics as metrics. Everyone else keeps today's behaviour, where the raw `pmm-otel` Grafana datasource stays Admin-only.

**Architecture** ([analysis §7](../specs/2026-10-10-opentelemetry-analysis.md#7-lbac)):
- **Enforcement:** a PMM query API, not the Grafana datasource.
  - The auth server already sends the user's selectors in `X-Proxy-Filter`.
  - The API evaluates them against inventory labels, producing allowed `(service_id, node_id)` pairs and node ids.
  - It binds them as query parameters, and **fails closed**.
- **Ingest:** identity is checked, so a node can't claim to be another node.
- **UI:** a log explorer and a trace view in the PMM UI use the API.

**Spec:** analysis §7.1–§7.4 and F8, F9; the existing LBAC model (`managed/models/role_*`, `managed/services/grafana/auth_server.go:146-160,446-494`).

## Global Constraints

- Selector semantics match VictoriaMetrics:
  - matchers inside a role are ANDed, and roles are ORed;
  - a role with an empty filter means full access;
  - a missing label equals `""`;
  - regexes are anchored.
- **Fail closed.** A non-Admin request without the header is denied, a selector that doesn't parse is denied, and service-account tokens get an explicit decision.
- No raw SQL from clients. Every filter is a bound parameter.
- Admins are not exempt from LBAC, consistent with today.

## Review Focus

1. **A role `{environment="prod"}` and a node with no `environment` label.** Expected: that node's OS logs are hidden, as its node metrics are.
2. **A service deleted after its logs were stored.** Expected: hidden from restricted users, visible to full-access users.
3. **A record whose `pmm.service_id` belongs to a service on another node** (a spoofed record). Expected: it is never shown to users allowed only on that service, because the pair predicate requires the service's own node.
4. **A trace that crosses an allowed and a forbidden service.** Expected: only the allowed spans are returned, and the response says the trace is partial.
5. **A role edited while a user has the explorer open.** Expected: the next request applies the new role. Nothing is cached beyond the inventory TTL.

---

### Task 0: ADR-17, LBAC for OTel data

- **Files:** `docs/adr/ADR-17-otel-lbac.md`, written in the format of PMM-15590's ADRs. It records options A–D from analysis §7.4 and the decision.
- **Exit:** accepted by the PMM architects and the security reviewer named in PMM-15591.

### Task 1: Shared selector evaluator

- **Files:**
  - `managed/utils/lbac/selectors.go`
  - `managed/utils/lbac/selectors_test.go`
- **Produces:**
  - `func Parse(filters []string) (Selectors, error)`
  - `func (s Selectors) Allows(labels map[string]string) bool`
  - `func (s Selectors) FullAccess() bool`
  - It uses the Prometheus parser that is already a dependency.
- **Tests:**
  - `TestAllowsAnchoredRegex` (`environment=~"prod"` doesn't match `preprod`);
  - `TestAllowsMissingLabelIsEmpty` (`{az!="x"}` matches a missing `az`);
  - `TestRolesOR`, `TestMatchersAND`;
  - `TestEmptyFilterIsFullAccess`;
  - `TestParseErrorFails`.
- QAN can adopt this package later, as a separate ticket.

### Task 2: Verified node identity at ingest

- **Files:**
  - `managed/services/grafana/auth_server.go`: for `/otlp/` requests carrying a per-node service-account token, set the `X-PMM-Node-Id` response header. Resolve the node from the service account the token belongs to, recording that mapping at registration if it isn't stored yet (`client.go:657-674`).
  - `pmm.conf` `/otlp/`: `auth_request_set $pmm_node_id $upstream_http_x_pmm_node_id; proxy_set_header X-PMM-Node-Id $pmm_node_id;`. Any client-sent header of that name is dropped.
  - `managed/services/otel/server_config.go`: OTLP HTTP `include_metadata: true`, plus a processor that upserts `pmm.node_id` from `metadata.x-pmm-node-id` when it is present, and otherwise sets `pmm.identity_verified=false`.
- **Spike first:** confirm in the pinned collector which processor can read client metadata into resource attributes: `resource` with `from_context`, or `transform` with OTTL request context. Record the result in the ADR.
- **Tests:**
  - an auth server unit test for the header;
  - a contract test: a record claiming `pmm.node_id=other`, sent with node A's header, is stored with node A's id.

### Task 3: `LogsService.Search` with an ID allow-list

- **Files:**
  - `api/logs/v1/logs.proto`: `Search` at `POST /v1/logs:search`, with `start`, `end`, `node_ids`, `service_ids`, `apps`, `severities`, `text`, `limit` (at most 1000) and `cursor`
  - `managed/services/logs/search.go` and its tests
  - `managed/services/grafana/auth_server.go`: rule `"/v1/logs": viewer`, and `/v1/logs` added to `lbacPrefixes`
  - `pmm.conf`: a `/v1/logs` location that sets `X-Proxy-Filter`, like `/v1/qan` (:339-344)
- **Produces:**
  - `func AllowedIDs(q *reform.Querier, s lbac.Selectors) (pairs []ServiceNode, nodes []string, err error)`, cached on the inventory TTL;
  - the predicate `(PmmServiceId, PmmNodeId) IN {pairs} OR (PmmServiceId = '' AND PmmNodeId IN {nodes})`, bound through clickhouse-go parameters.
- **Tests:**
  - `TestSearchFailsClosedWithoutHeader`;
  - `TestSearchDeniesBadSelector`;
  - `TestSearchHidesSpoofedServiceRow`;
  - `TestSearchNodeLogsNeedNodeLabel`;
  - an api-test with two roles and two users.

### Task 4: `TracesService.Search` and `GetTrace`

- The same predicate, applied to each span. `GetTrace` returns `partial: true` when spans were filtered out.
- **Tests:** `TestGetTracePartial`, plus the fail-closed tests from Task 3.

### Task 5: Log explorer and trace view in the PMM UI

- **Files:**
  - `ui/apps/pmm/src/pages/logs/*`
  - `ui/apps/pmm/src/pages/traces/*`
  - `api/logs.ts`
  - `hooks/api/useLogs.ts`
  - the routes in `router.tsx`
- **Content:**
  - filters for node, service, app, severity, text and time range;
  - links to service dashboards and QAN;
  - a "partial trace" notice.
- **Tests:** Vitest for the filters and the partial notice. A pmm-qa e2e test with a restricted user.

### Task 6: Security-negative QA and docs

- **QA** (pmm-qa, PMM-15578 style):
  - a restricted user sees only its services' logs and traces;
  - an unparsable role denies;
  - a spoofed node id is corrected.
- **Docs:** the LBAC page gains a "Logs and traces" section that explains the semantics in Review Focus 1, 2 and 4.
