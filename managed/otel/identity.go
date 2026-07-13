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

import (
	"strings"
)

// Phase1 identity: resource attribute keys required for service-map rollups (see docs/internal/ebpf-otel-identity-v1.md).
const (
	AttrServiceName      = "service.name"
	AttrPMMNodeID        = "pmm.node_id"
	AttrPMMAgentID       = "pmm.agent_id"
	AttrNetPeerName      = "net.peer.name"
	AttrNetPeerIP        = "net.peer.ip"
	AttrNetPeerPort      = "net.peer.port"
	AttrDBSystem         = "db.system"
	AttrPMMComponentRole = "pmm.component_role"
	AttrPMMMapEdgeTarget = "pmm.map_edge_target"
	AttrErrorType        = "error.type"
)

// IdentityCheck reports whether resource attributes satisfy Phase 1 map eligibility.
type IdentityCheck struct {
	OK       bool
	Missing  []string
	HasPeer  bool
	HasDBSys bool
}

// CheckPhase1ResourceIdentity validates resource attributes (string map as stored in CH Map or OTLP).
func CheckPhase1ResourceIdentity(attrs map[string]string) IdentityCheck {
	var missing []string
	add := func(ok bool, name string) {
		if !ok {
			missing = append(missing, name)
		}
	}

	add(strings.TrimSpace(attrs[AttrServiceName]) != "", AttrServiceName)
	add(strings.TrimSpace(attrs[AttrPMMNodeID]) != "", AttrPMMNodeID)
	add(strings.TrimSpace(attrs[AttrPMMAgentID]) != "", AttrPMMAgentID)

	hasPeer := strings.TrimSpace(attrs[AttrNetPeerName]) != "" ||
		(strings.TrimSpace(attrs[AttrNetPeerIP]) != "" && strings.TrimSpace(attrs[AttrNetPeerPort]) != "")
	add(hasPeer, "net.peer.name_or_ip_port")
	role := strings.TrimSpace(attrs[AttrPMMComponentRole])
	add(role != "", AttrPMMComponentRole)

	dbSys := strings.TrimSpace(attrs[AttrDBSystem])
	hasDB := dbSys != ""
	// For database role, db.system should be set; for other roles it is still recommended.
	if strings.EqualFold(role, "database") {
		add(hasDB, AttrDBSystem)
	}

	return IdentityCheck{
		OK:       len(missing) == 0,
		Missing:  missing,
		HasPeer:  hasPeer,
		HasDBSys: hasDB,
	}
}
