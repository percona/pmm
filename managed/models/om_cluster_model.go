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

package models

import (
	"time"

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// OmClusterKind is the topology of a managed cluster.
type OmClusterKind string

// Managed cluster kinds.
const (
	OmClusterReplicaSet = OmClusterKind("replica_set")
	OmClusterSharded    = OmClusterKind("sharded")
)

// OmClusterOrigin says how PMM came to manage a cluster; it is set once.
type OmClusterOrigin string

// Managed cluster origins.
const (
	OmClusterInstalled = OmClusterOrigin("installed")
	OmClusterImported  = OmClusterOrigin("imported")
)

// OmClusterStatus is a managed cluster's lifecycle status.
type OmClusterStatus string

// Managed cluster lifecycle statuses.
const (
	OmClusterInstalling = OmClusterStatus("installing")
	OmClusterImporting  = OmClusterStatus("importing")
	OmClusterMonitored  = OmClusterStatus("monitored")
	OmClusterManaged    = OmClusterStatus("managed")
	OmClusterRemoving   = OmClusterStatus("removing")
	OmClusterRemoved    = OmClusterStatus("removed")
)

// OmCluster is the record Operations acts on: a managed MongoDB cluster.
//
//reform:om_clusters
type OmCluster struct {
	ID             string          `reform:"id,pk"`
	Name           string          `reform:"name"`
	Kind           OmClusterKind   `reform:"kind"`
	Origin         OmClusterOrigin `reform:"origin"`
	Status         OmClusterStatus `reform:"status"`
	MongoClusterID *string         `reform:"mongo_cluster_id"`
	StateVersion   *int32          `reform:"state_version"`
	CreatedBy      string          `reform:"created_by"`
	CreatedAt      time.Time       `reform:"created_at"`
	UpdatedAt      time.Time       `reform:"updated_at"`
	RemovedAt      *time.Time      `reform:"removed_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (c *OmCluster) BeforeInsert() error {
	now := Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater.
func (c *OmCluster) BeforeUpdate() error {
	c.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder.
func (c *OmCluster) AfterFind() error {
	c.CreatedAt = c.CreatedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()
	if c.RemovedAt != nil {
		t := c.RemovedAt.UTC()
		c.RemovedAt = &t
	}
	return nil
}

// OmReplicaSetRole is a replica set's role within its cluster.
type OmReplicaSetRole string

// Replica set roles. A sharded cluster's config servers and every mongos are one domain.
const (
	OmReplicaSetRolePlain  = OmReplicaSetRole("replica_set")
	OmReplicaSetRoleShard  = OmReplicaSetRole("shard")
	OmReplicaSetRoleConfig = OmReplicaSetRole("config")
)

// OmReplicaSet is one replica set of a cluster and the authentication domain for its users.
//
//reform:om_replica_sets
type OmReplicaSet struct {
	ID           string           `reform:"id,pk"`
	ClusterID    string           `reform:"cluster_id"`
	Name         string           `reform:"name"`
	ReplicaSetID *string          `reform:"replica_set_id"`
	Role         OmReplicaSetRole `reform:"role"`
	CreatedAt    time.Time        `reform:"created_at"`
	RemovedAt    *time.Time       `reform:"removed_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (r *OmReplicaSet) BeforeInsert() error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = Now()
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (r *OmReplicaSet) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	if r.RemovedAt != nil {
		t := r.RemovedAt.UTC()
		r.RemovedAt = &t
	}
	return nil
}

// OmMemberProcess is the MongoDB process a member runs.
type OmMemberProcess string

// Member processes.
const (
	OmMemberMongod = OmMemberProcess("mongod")
	OmMemberMongos = OmMemberProcess("mongos")
)

// OmMemberState is a member's membership state in the accepted topology.
type OmMemberState string

// Member states.
const (
	OmMemberPendingAdd    = OmMemberState("pending_add")
	OmMemberAccepted      = OmMemberState("accepted")
	OmMemberPendingRemove = OmMemberState("pending_remove")
)

// OmClusterMember is a member keyed by MongoDB's identity, with nullable PMM links.
//
//reform:om_cluster_members
type OmClusterMember struct {
	ID                 string          `reform:"id,pk"`
	ClusterID          string          `reform:"cluster_id"`
	ReplicaSetRef      *string         `reform:"replica_set_ref"`
	Process            OmMemberProcess `reform:"process"`
	Host               string          `reform:"host"`
	RSMemberID         *int32          `reform:"rs_member_id"`
	State              OmMemberState   `reform:"state"`
	NodeID             *string         `reform:"node_id"`
	ServiceID          *string         `reform:"service_id"`
	ExporterAgentID    *string         `reform:"exporter_agent_id"`
	AddedByOperation   string          `reform:"added_by_operation"`
	RemovedByOperation *string         `reform:"removed_by_operation"`
	CreatedAt          time.Time       `reform:"created_at"`
	RemovedAt          *time.Time      `reform:"removed_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (m *OmClusterMember) BeforeInsert() error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = Now()
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (m *OmClusterMember) AfterFind() error {
	m.CreatedAt = m.CreatedAt.UTC()
	if m.RemovedAt != nil {
		t := m.RemovedAt.UTC()
		m.RemovedAt = &t
	}
	return nil
}

// OmClusterState is one append-only accepted-state version (see AppendClusterState).
//
//reform:om_cluster_states
type OmClusterState struct {
	ClusterID     string    `reform:"cluster_id"`
	Version       int32     `reform:"version"`
	SchemaVersion int32     `reform:"schema_version"`
	Document      []byte    `reform:"document"`
	OperationID   string    `reform:"operation_id"`
	CreatedAt     time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (s *OmClusterState) BeforeInsert() error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = Now()
	}
	return nil
}

// AfterFind implements reform.AfterFinder.
func (s *OmClusterState) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*OmCluster)(nil)
	_ reform.BeforeUpdater  = (*OmCluster)(nil)
	_ reform.AfterFinder    = (*OmCluster)(nil)
	_ reform.BeforeInserter = (*OmReplicaSet)(nil)
	_ reform.AfterFinder    = (*OmReplicaSet)(nil)
	_ reform.BeforeInserter = (*OmClusterMember)(nil)
	_ reform.AfterFinder    = (*OmClusterMember)(nil)
	_ reform.BeforeInserter = (*OmClusterState)(nil)
	_ reform.AfterFinder    = (*OmClusterState)(nil)
)
