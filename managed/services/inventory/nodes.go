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

package inventory

import (
	"context"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// NodesService works with inventory API Nodes.
type NodesService struct {
	db    *reform.DB
	r     agentsRegistry
	state agentsStateUpdater
	vmdb  prometheusService
}

// NewNodesService returns Inventory API handler for managing Nodes.
func NewNodesService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService) *NodesService {
	return &NodesService{
		db:    db,
		r:     r,
		state: state,
		vmdb:  vmdb,
	}
}

// List returns a list of all Nodes.
//
//nolint:unparam
func (s *NodesService) List(ctx context.Context, filters models.NodeFilters) ([]inventorypb.Node, error) {
	var nodes []*models.Node
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		nodes, err = models.FindNodes(tx.Querier, filters)
		return err
	})
	if e != nil {
		return nil, e
	}

	res := make([]inventorypb.Node, len(nodes))
	for i, n := range nodes {
		res[i], e = services.ToAPINode(n)
		if e != nil {
			return nil, e
		}
	}
	return res, nil
}

// Get returns a single Node by ID.
//
//nolint:unparam
func (s *NodesService) Get(ctx context.Context, req *inventorypb.GetNodeRequest) (inventorypb.Node, error) {
	modelNode := &models.Node{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		modelNode, err = models.FindNodeByID(tx.Querier, req.NodeId)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	node, err := services.ToAPINode(modelNode)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// AddGenericNode adds Generic Node.
//
//nolint:unparam
func (s *NodesService) AddGenericNode(ctx context.Context, req *inventorypb.AddGenericNodeRequest) (*inventorypb.GenericNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		MachineID:    pointer.ToStringOrNil(req.MachineId),
		Distro:       req.Distro,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		node, err = models.CreateNode(tx.Querier, models.GenericNodeType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	invNode, err := services.ToAPINode(node)
	if err != nil {
		return nil, err
	}

	return invNode.(*inventorypb.GenericNode), nil
}

// AddContainerNode adds Container Node.
//
//nolint:unparam
func (s *NodesService) AddContainerNode(ctx context.Context, req *inventorypb.AddContainerNodeRequest) (*inventorypb.ContainerNode, error) {
	params := &models.CreateNodeParams{
		NodeName:      req.NodeName,
		Address:       req.Address,
		MachineID:     pointer.ToStringOrNil(req.MachineId),
		ContainerID:   pointer.ToStringOrNil(req.ContainerId),
		ContainerName: pointer.ToStringOrNil(req.ContainerName),
		NodeModel:     req.NodeModel,
		Region:        pointer.ToStringOrNil(req.Region),
		AZ:            req.Az,
		CustomLabels:  req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		node, err = models.CreateNode(tx.Querier, models.ContainerNodeType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	invNode, err := services.ToAPINode(node)
	if err != nil {
		return nil, err
	}

	return invNode.(*inventorypb.ContainerNode), nil
}

// AddRemoteNode adds Remote Node.
//
//nolint:unparam
func (s *NodesService) AddRemoteNode(ctx context.Context, req *inventorypb.AddRemoteNodeRequest) (*inventorypb.RemoteNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		node, err = models.CreateNode(tx.Querier, models.RemoteNodeType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	invNode, err := services.ToAPINode(node)
	if err != nil {
		return nil, err
	}

	return invNode.(*inventorypb.RemoteNode), nil
}

// AddRemoteRDSNode adds a new RDS node
//
//nolint:unparam
func (s *NodesService) AddRemoteRDSNode(ctx context.Context, req *inventorypb.AddRemoteRDSNodeRequest) (*inventorypb.RemoteRDSNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		node, err = models.CreateNode(tx.Querier, models.RemoteRDSNodeType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	invNode, err := services.ToAPINode(node)
	if err != nil {
		return nil, err
	}

	return invNode.(*inventorypb.RemoteRDSNode), nil
}

// AddRemoteAzureDatabaseNode adds a new Azure database node
//
//nolint:unparam,dupl
func (s *NodesService) AddRemoteAzureDatabaseNode(ctx context.Context, req *inventorypb.AddRemoteAzureDatabaseNodeRequest) (*inventorypb.RemoteAzureDatabaseNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		node, err = models.CreateNode(tx.Querier, models.RemoteAzureDatabaseNodeType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	invNode, err := services.ToAPINode(node)
	if err != nil {
		return nil, err
	}

	return invNode.(*inventorypb.RemoteAzureDatabaseNode), nil
}

// Remove removes Node without any Agents and Services.
// Removes Node with the Agents and Services if force == true.
// Returns an error if force == false and Node has Agents or Services.
func (s *NodesService) Remove(ctx context.Context, id string, force bool) error {
	idsToKick := make(map[string]struct{})
	idsToSetState := make(map[string]struct{})

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if force {
			mode = models.RemoveCascade

			agents, err := models.FindPMMAgentsRunningOnNode(tx.Querier, id)
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				idsToKick[a.AgentID] = struct{}{}
			}

			agents, err = models.FindAgents(tx.Querier, models.AgentFilters{NodeID: id})
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				if a.PMMAgentID != nil {
					idsToSetState[pointer.GetString(a.PMMAgentID)] = struct{}{}
				}
			}

			agents, err = models.FindPMMAgentsForServicesOnNode(tx.Querier, id)
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				idsToSetState[a.AgentID] = struct{}{}
			}
		}
		return models.RemoveNode(tx.Querier, id, mode)
	}); e != nil {
		return e
	}

	for id := range idsToSetState {
		s.state.RequestStateUpdate(ctx, id)
	}
	for id := range idsToKick {
		s.r.Kick(ctx, id)
	}

	if force {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		s.vmdb.RequestConfigurationUpdate()
	}

	return nil
}
