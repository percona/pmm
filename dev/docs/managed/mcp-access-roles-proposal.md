# Proposal: PMM Access-Role permissions for the MCP endpoint (not implemented)

Status: design proposal accompanying the MCP server branch (`managed/services/mcp`).
Deliberately **not** implemented in that branch (decision D4 in
`managed/services/mcp/DECISIONS.md`): the branch ships two `methodRules`
entries that make inventory *reads* Viewer-level, which is enough for a
least-privilege MCP token today. This document records the longer-term
"new role" design so that Percona can decide the direction.

## Problem

1. Grafana OSS has exactly three organisation roles (Viewer, Editor, Admin) and
   PMM does not fork that part, so a fourth Grafana role is not an option.
   Today the only way to let a non-admin token list services is to widen the
   Viewer (or Editor) role for `GET /v1/inventory/{services,nodes}`.
2. PMM's own Access Roles (`roles` table: `title`, `description`, `filter`) are
   label-filter-only: they carry no permissions, so they cannot express
   "may read the inventory" or "may use `/mcp`".
3. Service-account tokens are resolved by `getRoleForServiceToken` with
   `userID: 0`, so PMM roles cannot be assigned to them and
   `maybeAddLBACFilters` skips them (it treats `userID <= 0` as anonymous).
   LBAC therefore never applies to tokens, including MCP callers.

## Design

### Model

- Add `permissions text[]` to `roles` with a forward-only migration in
  `managed/models/database.go`; extend the reform model and helpers.
- Initial permission vocabulary: `inventory:read`, `mcp:use`.
- Seed a built-in, non-deletable role **"Inventory Reader"** (`inventory:read`,
  `mcp:use`, no filter) so an admin has a one-click choice.

### API

- Additive `repeated string permissions` on `Role` in
  `api/accesscontrol/v1beta1/accesscontrol.proto` (`buf breaking`-safe),
  then `make gen`.

### Enforcement

- A third map in `managed/services/grafana/auth_server.go`,
  `permissionRules`, e.g. `"GET /v1/inventory/services": "inventory:read"`,
  `"/mcp": "mcp:use"`.
- In `authenticate`, after the Grafana-role check fails: if access control is
  enabled and the caller's PMM roles grant the required permission, allow.
  Everything else is untouched, so with access control disabled the behaviour
  is exactly today's.

### Service accounts as principals (the blocker)

- Resolve the service account's Grafana user ID when a token authenticates.
  Grafana's `/api/auth/serviceaccount` response in the Percona fork needs
  checking for an `id` / `userId` field; otherwise
  `GET /api/serviceaccounts/search?query=<name>` gives it.
- This one fix also makes LBAC work for tokens, which is worth doing regardless
  of MCP.

### UI

- A "Permissions" checkbox group on the Access Roles editor (`ui/apps/pmm`).
- A role picker on the service-account page lives in the Grafana fork; a
  lighter alternative is to document `POST /v1/accesscontrol/roles:assign` for
  token principals and add a PMM-side "Assign PMM role" action later.

## Effort and risk

Roughly a week (managed + proto + UI) plus Percona's review of the RBAC
direction. It touches areas where Percona is likely to have a roadmap, which
is why the MCP branch ships the small `methodRules` change (works with or
without this proposal) and presents this document instead of blocking on it.

## Interaction with the MCP branch

- If this proposal lands, the two `methodRules` entries can stay (Viewer keeps
  read access) or be replaced by `permissionRules` entries; the MCP code does
  not change either way, because it authorizes nothing itself: every tool call
  is re-authorized by nginx `auth_request` on the backing REST path.
- Once tokens carry a user ID, LBAC filters apply to the MCP's metrics queries
  through the `/graph/api/datasources/proxy/uid/` prefix the branch added to
  `lbacPrefixes`, and to QAN through the existing `/v1/qan/` prefix.
