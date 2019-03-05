// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// NodesForAgent returns all Nodes for which Agent with given ID provides insights.
func NodesForAgent(q *reform.Querier, agentID string) ([]*Node, error) {
	structs, err := q.FindAllFrom(AgentNodeView, "agent_id", agentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Node IDs")
	}

	nodeIDs := make([]interface{}, len(structs))
	for i, s := range structs {
		nodeIDs[i] = s.(*AgentNode).NodeID
	}
	if len(nodeIDs) == 0 {
		return []*Node{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(nodeIDs)), ", ")
	tail := fmt.Sprintf("WHERE node_id IN (%s) ORDER BY node_id", p) //nolint:gosec
	structs, err = q.SelectAllFrom(NodeTable, tail, nodeIDs...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Nodes")
	}

	res := make([]*Node, len(structs))
	for i, s := range structs {
		res[i] = s.(*Node)
	}
	return res, nil
}

//go:generate reform

// NodeType represents Node type as stored in database.
type NodeType string

// Node types.
const (
	GenericNodeType         NodeType = "generic"
	ContainerNodeType       NodeType = "container"
	RemoteNodeType          NodeType = "remote"
	RemoteAmazonRDSNodeType NodeType = "remote-amazon-rds"
)

// PMMServerNodeID is a special Node ID representing PMM Server Node.
const PMMServerNodeID string = "pmm-server"

// Node represents Node as stored in database.
//reform:nodes
type Node struct {
	NodeID       string    `reform:"node_id,pk"`
	NodeType     NodeType  `reform:"node_type"`
	NodeName     string    `reform:"node_name"`
	MachineID    *string   `reform:"machine_id"` // nil means "unknown"; non-nil value must be unique
	CustomLabels []byte    `reform:"custom_labels"`
	Address      string    `reform:"address"` // also Remote instance
	CreatedAt    time.Time `reform:"created_at"`
	// UpdatedAt time.Time `reform:"updated_at"`

	Distro        string `reform:"distro"`
	DistroVersion string `reform:"distro_version"`

	DockerContainerID   *string `reform:"docker_container_id"` // nil means "unknown"; non-nil value must be unique
	DockerContainerName string  `reform:"docker_container_name"`

	Region *string `reform:"region"` // nil means "not Remote"; non-nil value must be unique in combination with instance/address
}

// BeforeInsert implements reform.BeforeInserter interface.
//nolint:unparam
func (s *Node) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	// s.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//nolint:unparam
func (s *Node) BeforeUpdate() error {
	// now := Now()
	// s.UpdatedAt = now
	return nil
}

// AfterFind implements reform.AfterFinder interface.
//nolint:unparam
func (s *Node) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	// s.UpdatedAt = s.UpdatedAt.UTC()
	return nil
}

// GetCustomLabels decodes custom labels.
func (s *Node) GetCustomLabels() (map[string]string, error) {
	if len(s.CustomLabels) == 0 {
		return nil, nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(s.CustomLabels, &m); err != nil {
		return nil, errors.Wrap(err, "failed to decode custom labels")
	}
	return m, nil
}

// SetCustomLabels encodes custom labels.
func (s *Node) SetCustomLabels(m map[string]string) error {
	if len(m) == 0 {
		s.CustomLabels = nil
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "failed to encode custom labels")
	}
	s.CustomLabels = b
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*Node)(nil)
	_ reform.BeforeUpdater  = (*Node)(nil)
	_ reform.AfterFinder    = (*Node)(nil)
)
