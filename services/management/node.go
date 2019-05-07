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

package management

import (
	"context"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"

	// FIXME Refactor, as service shouldn't depend on other service in one abstraction level.
	// https://jira.percona.com/browse/PMM-3541
	// See also main_test.go
	"github.com/percona/pmm-managed/services/inventory"
)

var (
	errNodeNotFound  = errors.New("node not found")
	errAgentNotFound = errors.New("agent not found")
)

// NodeService represents service for working with nodes.
type NodeService struct {
	db       *reform.DB
	registry registry
}

// NewNodeService creates NodeService instance.
func NewNodeService(db *reform.DB, registry registry) *NodeService {
	return &NodeService{
		db:       db,
		registry: registry,
	}
}

// Register do registration of the new node.
func (s *NodeService) Register(ctx context.Context, req *managementpb.RegisterNodeRequest) (res *managementpb.RegisterNodeResponse, err error) {
	res = new(managementpb.RegisterNodeResponse)

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		node, err := s.findNodeByName(tx.Querier, req.NodeName)
		switch err {
		case nil:
			params := &models.UpdateNodeParams{
				Address:      req.Address,
				MachineID:    req.MachineId,
				CustomLabels: req.CustomLabels,
			}
			node, err = models.UpdateNode(tx.Querier, node.NodeID, params)
			if err != nil {
				return err
			}
		case errNodeNotFound:
			node, err = s.createNewNode(tx.Querier, req)
			if err != nil {
				return err
			}
		default:
			return err
		}

		if err := s.addNodeToResponse(node, res); err != nil {
			return err
		}

		pmmAgent, err := s.findPmmAgentByNodeID(tx.Querier, node.NodeID)
		switch err {
		case errAgentNotFound:
			pmmAgent, err = models.AgentAddPmmAgent(tx.Querier, node.NodeID, nil)
			if err != nil {
				return err
			}
		case nil:
			// noop
		default:
			return err
		}

		if err := s.addPmmAgentToResponse(tx.Querier, pmmAgent, res); err != nil {
			return err
		}

		_, err = s.findNodeExporterByPmmAgentID(tx.Querier, pmmAgent.AgentID)
		switch err {
		case errAgentNotFound:
			_, err := models.AgentAddNodeExporter(tx.Querier, pmmAgent.AgentID, nil)
			if err != nil {
				return err
			}
		case nil:
			// noop
		default:
			return err
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.registry.SendSetStateRequest(ctx, res.PmmAgent.AgentId)

	return res, nil
}

func (s *NodeService) createNewNode(q *reform.Querier, req *managementpb.RegisterNodeRequest) (*models.Node, error) {
	var nodeType models.NodeType
	switch req.NodeType {
	case inventorypb.NodeType_GENERIC_NODE:
		nodeType = models.GenericNodeType
	case inventorypb.NodeType_CONTAINER_NODE:
		nodeType = models.ContainerNodeType
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported node type")
	}

	params := &models.CreateNodeParams{
		NodeName:      req.NodeName,
		MachineID:     pointer.ToStringOrNil(req.MachineId),
		Distro:        req.Distro,
		NodeModel:     "", // TODO
		AZ:            "", // TODO
		ContainerID:   pointer.ToStringOrNil(req.ContainerId),
		ContainerName: pointer.ToStringOrNil(req.ContainerName),
		CustomLabels:  req.CustomLabels,
		Address:       req.Address,
		Region:        nil, // TODO
	}
	node, err := models.CreateNode(q, nodeType, params)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (s *NodeService) findNodeByName(q *reform.Querier, name string) (*models.Node, error) {
	nodes, err := models.FindAllNodes(q)
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		if n.NodeName == name {
			return n, nil
		}
	}

	return nil, errNodeNotFound
}

func (s *NodeService) findPmmAgentByNodeID(q *reform.Querier, nodeID string) (pmmAgent *models.Agent, err error) {
	agents, err := models.AgentFindAll(q)
	if err != nil {
		return nil, err
	}

	for _, a := range agents {
		if pointer.GetString(a.RunsOnNodeID) == nodeID {
			return a, nil
		}
	}

	return pmmAgent, errAgentNotFound
}

func (s *NodeService) findNodeExporterByPmmAgentID(q *reform.Querier, pmmAgentID string) (nodeExporter *inventorypb.NodeExporter, err error) {
	agents, err := models.AgentsRunningByPMMAgent(q, pmmAgentID)
	if err != nil {
		return nil, err
	}

	for _, a := range agents {
		if pointer.GetString(a.PMMAgentID) == pmmAgentID {
			invAgent, err := inventory.ToInventoryAgent(q, a, s.registry)
			if err != nil {
				return nodeExporter, err
			}
			nodeExporter = invAgent.(*inventorypb.NodeExporter)
			return nodeExporter, nil
		}
	}

	return nodeExporter, errAgentNotFound
}

func (s *NodeService) addNodeToResponse(model *models.Node, res *managementpb.RegisterNodeResponse) error {
	node, err := inventory.ToInventoryNode(model)
	if err != nil {
		return err
	}

	switch n := node.(type) {
	case *inventorypb.GenericNode:
		res.GenericNode = n
	case *inventorypb.ContainerNode:
		res.ContainerNode = n
	default:
		return status.Error(codes.InvalidArgument, "unsupported node type")
	}

	return nil
}

func (s *NodeService) addPmmAgentToResponse(q *reform.Querier, model *models.Agent, res *managementpb.RegisterNodeResponse) error {
	invAgent, err := inventory.ToInventoryAgent(q, model, s.registry)
	if err != nil {
		return err
	}
	res.PmmAgent = invAgent.(*inventorypb.PMMAgent)
	return nil
}
