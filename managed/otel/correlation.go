// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package otel

// ProxyComponent identifies proxy / router legs for Phase 1 topology stitching (see docs/internal/ebpf-proxy-ha-correlation.md).
type ProxyComponent string

const (
	ProxyHAProxy   ProxyComponent = "haproxy"
	ProxyProxySQL  ProxyComponent = "proxysql"
	ProxyPgBouncer ProxyComponent = "pgbouncer"
	ProxyMongos    ProxyComponent = "mongos"
)

// OrchestratorKind marks control-plane enrichers (Patroni, etc.).
type OrchestratorKind string

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
