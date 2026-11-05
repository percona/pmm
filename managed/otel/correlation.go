// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package otel

// ProxyComponent identifies proxy / router legs for Phase 1 topology stitching (see docs/internal/ebpf-proxy-ha-correlation.md).
type ProxyComponent string

// Proxy component identifiers used in topology stitching.
const (
	ProxyHAProxy   ProxyComponent = "haproxy"
	ProxyProxySQL  ProxyComponent = "proxysql"
	ProxyPgBouncer ProxyComponent = "pgbouncer"
	ProxyMongos    ProxyComponent = "mongos"
)

// OrchestratorKind marks control-plane enrichers (Patroni, etc.).
type OrchestratorKind string

// Orchestrator kinds known to PMM.
const (
	OrchestratorPatroni OrchestratorKind = "patroni"
)

// AnnotateFailoverWindow describes a coarse failover interval attached to edges or nodes in rollups (MVP depth).
type AnnotateFailoverWindow struct {
	StartUnix int64
	EndUnix   int64
	Kind      OrchestratorKind
	Note      string
}
