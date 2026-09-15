// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package mcp implements PMM Server's Model Context Protocol (MCP) endpoint.
//
// The endpoint is served at /mcp over MCP Streamable HTTP and exposes a small,
// read-only tool surface (inventory, Query Analytics, EXPLAIN, schema and
// configuration) to any MCP client such as Claude Code or Cursor.
//
// Every tool call goes back out through PMM's public REST API on the nginx
// loopback (http://127.0.0.1:8080 by default), forwarding the caller's
// Authorization and Cookie headers verbatim. The package never reads
// ClickHouse, VictoriaMetrics or PostgreSQL directly and never calls
// pmm-managed's internal services: the tools therefore inherit PMM's
// per-caller RBAC (and LBAC where nginx injects it) and keep working across
// backend re-architecture as long as the /v1/ API contract holds.
//
// Authentication is PMM's own: nginx protects /mcp with the existing
// auth_request flow (minimum role Viewer, see managed/services/grafana), so
// the handler itself never answers HTTP 401 - that status is reserved for
// auth_request. Authorization failures on backing calls surface as MCP tool
// errors (isError: true, code "unauthorized").
package mcp
