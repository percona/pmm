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

package agents

import (
	"sync"

	"github.com/google/uuid"

	"github.com/percona/pmm/api/inventory"
)

// Store is a temporary store for nodes, services, and agents.
// FIXME It should be replaced with database.
type Store struct {
	m            sync.Mutex
	nodes        map[uint32]*inventory.BareMetalNode  // ID => node
	exporters    map[uint32]*inventory.MySQLdExporter // ID => exporter
	newExporters chan *inventory.MySQLdExporter
	lastNodeID   uint32
	lastAgentID  uint32
}

func NewStore() *Store {
	return &Store{
		nodes:        make(map[uint32]*inventory.BareMetalNode),
		exporters:    make(map[uint32]*inventory.MySQLdExporter),
		newExporters: make(chan *inventory.MySQLdExporter),
	}
}

func (s *Store) RegisterAgent() string {
	uuid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return uuid.String()
}

func (s *Store) AddNode(req *inventory.AddNodeRequest) *inventory.AddNodeResponse {
	s.m.Lock()
	defer s.m.Unlock()

	s.lastNodeID++
	id := s.lastNodeID
	node := &inventory.BareMetalNode{
		Id:       id,
		Name:     req.Name,
		Hostname: req.Hostname,
	}
	s.nodes[id] = node
	return &inventory.AddNodeResponse{
		Node: &inventory.AddNodeResponse_BareMetal{
			BareMetal: node,
		},
	}
}

func (s *Store) AddMySQLdExporter(req *inventory.AddMySQLdExporterAgentRequest) *inventory.AddMySQLdExporterAgentResponse {
	s.m.Lock()
	defer s.m.Unlock()

	s.lastAgentID++
	id := s.lastAgentID
	exporter := &inventory.MySQLdExporter{
		Id:           id,
		RunsOnNodeId: req.RunsOnNodeId,
		Username:     req.Username,
		Password:     req.Password,
		ListenPort:   12345,
	}
	s.exporters[id] = exporter

	s.newExporters <- exporter

	return &inventory.AddMySQLdExporterAgentResponse{
		MysqldExporter: exporter,
	}
}

func (s *Store) NewExporters() <-chan *inventory.MySQLdExporter {
	return s.newExporters
}
