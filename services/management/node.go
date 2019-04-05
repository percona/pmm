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
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/percona/pmm/api/managementpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/inventory"
)

// NodeService represents service for working with nodes.
type NodeService struct {
	db *reform.DB
	ns *inventory.NodesService
	ag *inventory.AgentsService
}

// NewNodeService creates NodeService instance.
func NewNodeService(db *reform.DB, ns *inventory.NodesService, ag *inventory.AgentsService) *NodeService {
	return &NodeService{
		db: db,
		ns: ns,
		ag: ag,
	}
}

// Register do registration of the new node.
func (s *NodeService) Register(ctx context.Context, req *managementpb.RegisterNodeRequest) (res *managementpb.RegisterNodeResponse, err error) {
	res = new(managementpb.RegisterNodeResponse)

	if e := s.db.InTransaction(func(tx *reform.TX) error {

		node, err := s.findNodeByName(ctx, tx.Querier, req.NodeName)
		switch err.(type) {
		case nodeNotFoundErr:
			node, err = s.createNewNode(ctx, tx.Querier, req)
			if err != nil {
				return err
			}
		case nil:
			params := &inventory.UpdateNodeParams{
				Address:      req.Address,
				MachineID:    req.MachineId,
				CustomLabels: req.CustomLabels,
			}
			node, err = s.ns.Update(ctx, tx.Querier, node.ID(), params)
			if err != nil {
				return err
			}
		default:
			return err
		}

		s.addNodeToResponse(node, res)

		pmmAgent, err := s.findPmmAgentByNodeID(ctx, tx.Querier, node.ID())
		switch err.(type) {
		case agentNotFoundErr:
			pmmParams := &inventorypb.AddPMMAgentRequest{RunsOnNodeId: node.ID()}
			pmmAgent, err = s.ag.AddPMMAgent(ctx, tx.Querier, pmmParams)
			if err != nil {
				return err
			}
		case nil:
			// noop
		default:
			return err
		}

		res.PmmAgent = pmmAgent

		_, err = s.findNodeExporterByPmmAgentID(ctx, tx.Querier, pmmAgent.ID())
		switch err.(type) {
		case agentNotFoundErr:
			nExpParams := &inventorypb.AddNodeExporterRequest{PmmAgentId: pmmAgent.ID()}
			_, err = s.ag.AddNodeExporter(ctx, tx.Querier, nExpParams)
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

	return res, nil
}

func (s *NodeService) createNewNode(ctx context.Context, q *reform.Querier, req *managementpb.RegisterNodeRequest) (inventorypb.Node, error) {
	var nodeType models.NodeType
	switch req.NodeType {
	case inventorypb.NodeType_GENERIC_NODE:
		nodeType = models.GenericNodeType
	case inventorypb.NodeType_CONTAINER_NODE:
		nodeType = models.ContainerNodeType
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported node type")
	}

	params := &inventory.AddNodeParams{
		NodeType:            nodeType,
		NodeName:            req.NodeName,
		MachineID:           pointer.ToStringOrNil(req.MachineId),
		Distro:              pointer.ToStringOrNil(req.Distro),
		DistroVersion:       pointer.ToStringOrNil(req.DistroVersion),
		DockerContainerID:   pointer.ToStringOrNil(req.ContainerId),
		DockerContainerName: pointer.ToStringOrNil(req.ContainerName),
		CustomLabels:        req.CustomLabels,
		Address:             pointer.ToStringOrNil(req.Address),
		Region:              nil,
	}
	node, err := s.ns.Add(ctx, q, params)
	if err != nil {
		return node, err
	}

	return node, nil
}

func (s *NodeService) findNodeByName(ctx context.Context, q *reform.Querier, name string) (inventorypb.Node, error) {
	nodes, err := s.ns.List(ctx, q)
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		if n.Name() == name {
			return n, nil
		}
	}

	var nfErr nodeNotFoundErr = "node not found"
	return nil, nfErr
}

func (s *NodeService) findPmmAgentByNodeID(ctx context.Context, q *reform.Querier, nodeID string) (pmmAgent *inventorypb.PMMAgent, err error) {
	agents, err := s.ag.List(ctx, q, inventory.AgentFilters{})
	if err != nil {
		return nil, err
	}

	var ok bool
	for _, a := range agents {
		pmmAgent, ok = a.(*inventorypb.PMMAgent)
		if ok && pmmAgent.RunsOnNodeId == nodeID {
			return pmmAgent, nil
		}
	}

	var anfErr agentNotFoundErr = "agent not found"
	return pmmAgent, anfErr
}

func (s *NodeService) findNodeExporterByPmmAgentID(ctx context.Context, q *reform.Querier, pmmAgentID string) (nodeExporter *inventorypb.NodeExporter, err error) {
	agents, err := s.ag.List(ctx, q, inventory.AgentFilters{PMMAgentID: pmmAgentID})
	if err != nil {
		return nil, err
	}

	var ok bool
	for _, a := range agents {
		nodeExporter, ok = a.(*inventorypb.NodeExporter)
		if ok && nodeExporter.PmmAgentId == pmmAgentID {
			return nodeExporter, nil
		}
	}

	var anfErr agentNotFoundErr = "agent not found"
	return nodeExporter, anfErr
}

func (s *NodeService) addNodeToResponse(node inventorypb.Node, res *managementpb.RegisterNodeResponse) {
	switch n := node.(type) {
	case *inventorypb.GenericNode:
		res.GenericNode = n
	case *inventorypb.ContainerNode:
		res.ContainerNode = n
	}
}

type nodeNotFoundErr string

func (e nodeNotFoundErr) Error() string {
	return string(e)
}

type agentNotFoundErr string

func (e agentNotFoundErr) Error() string {
	return string(e)
}
