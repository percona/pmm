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
	"maps"
	"time"

	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// NodeType represents Node type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and VictoriaMetrics.
type NodeType string

// Node types (in the same order as in nodes.proto).
const (
	GenericNodeType             NodeType = "generic"
	ContainerNodeType           NodeType = "container"
	RemoteNodeType              NodeType = "remote"
	RemoteRDSNodeType           NodeType = "remote_rds"
	RemoteAzureDatabaseNodeType NodeType = "remote_azure_database"
)

// PMMServerNodeID is a special Node ID representing PMM Server Node.
const PMMServerNodeID = string("pmm-server") // A special ID reserved for PMM Server Node.

// Node represents Node as stored in database.
//
//reform:nodes
type Node struct {
	NodeID       string   `reform:"node_id,pk"`
	NodeType     NodeType `reform:"node_type"`
	NodeName     string   `reform:"node_name"`
	MachineID    *string  `reform:"machine_id"`
	Distro       string   `reform:"distro"`
	NodeModel    string   `reform:"node_model"`
	AZ           string   `reform:"az"`
	CustomLabels []byte   `reform:"custom_labels"`

	// Node address. Used to construct the endpoint for node_exporter.
	// For RemoteRDS Nodes contains DBInstanceIdentifier (not DbiResourceId; not endpoint - that's Service address).
	Address    string  `reform:"address"`
	InstanceId *string `reform:"instance_id"` // nil means "unknown"; non-nil value must be unique

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`

	ContainerID   *string `reform:"container_id"` // nil means "unknown"; non-nil value must be unique
	ContainerName *string `reform:"container_name"`

	Region *string `reform:"region"` // non-nil value must be unique in combination with instance/address
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *Node) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *Node) BeforeUpdate() error {
	s.UpdatedAt = Now()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *Node) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// GetCustomLabels decodes custom labels.
func (s *Node) GetCustomLabels() (map[string]string, error) {
	return getLabels(s.CustomLabels)
}

// SetCustomLabels encodes custom labels.
func (s *Node) SetCustomLabels(m map[string]string) error {
	return setLabels(m, &s.CustomLabels)
}

// UnifiedLabels returns combined standard and custom labels with empty labels removed.
func (s *Node) UnifiedLabels() (map[string]string, error) {
	custom, err := s.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	res := map[string]string{
		"node_id":        s.NodeID,
		"node_name":      s.NodeName,
		"node_type":      string(s.NodeType),
		"machine_id":     pointer.GetString(s.MachineID),
		"container_id":   pointer.GetString(s.ContainerID),
		"container_name": pointer.GetString(s.ContainerName),
		"node_model":     s.NodeModel,
		"region":         pointer.GetString(s.Region),
		"az":             s.AZ,
	}
	maps.Copy(res, custom)

	if err = prepareLabels(res, true); err != nil {
		return nil, err
	}
	return res, nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Node)(nil)
	_ reform.BeforeUpdater  = (*Node)(nil)
	_ reform.AfterFinder    = (*Node)(nil)
)
