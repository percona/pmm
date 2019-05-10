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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"

	// FIXME Refactor, as service shouldn't depend on other service in one abstraction level.
	// https://jira.percona.com/browse/PMM-3541
	// See also main_test.go
	"github.com/percona/pmm-managed/services/inventory"
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
func (s *NodeService) Register(ctx context.Context, req *managementpb.RegisterNodeRequest) (*managementpb.RegisterNodeResponse, error) {
	res := new(managementpb.RegisterNodeResponse)

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		node, err := models.FindNodeByName(tx.Querier, req.NodeName)
		switch status.Code(err) {
		case codes.OK:
			if !req.Reregister {
				return status.Errorf(codes.AlreadyExists, "Node with name %q already exists.", req.NodeName)
			}
			err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade)
		case codes.NotFound:
			err = nil
		}
		if err != nil {
			return err
		}

		var nodeType models.NodeType
		switch req.NodeType {
		case inventorypb.NodeType_GENERIC_NODE:
			nodeType = models.GenericNodeType
		case inventorypb.NodeType_CONTAINER_NODE:
			nodeType = models.ContainerNodeType
		default:
			return status.Errorf(codes.InvalidArgument, "Unsupported Node type %q.", req.NodeType)
		}
		node, err = models.CreateNode(tx.Querier, nodeType, &models.CreateNodeParams{
			NodeName:      req.NodeName,
			MachineID:     pointer.ToStringOrNil(req.MachineId),
			Distro:        req.Distro,
			NodeModel:     req.NodeModel,
			AZ:            req.Az,
			ContainerID:   pointer.ToStringOrNil(req.ContainerId),
			ContainerName: pointer.ToStringOrNil(req.ContainerName),
			CustomLabels:  req.CustomLabels,
			Address:       req.Address,
			Region:        pointer.ToStringOrNil(req.Region),
		})
		if err != nil {
			return err
		}

		n, err := inventory.ToInventoryNode(node)
		if err != nil {
			return err
		}
		switch n := n.(type) {
		case *inventorypb.GenericNode:
			res.GenericNode = n
		case *inventorypb.ContainerNode:
			res.ContainerNode = n
		default:
			return status.Errorf(codes.InvalidArgument, "Unsupported Node type %q.", req.NodeType)
		}

		pmmAgent, err := models.AgentAddPmmAgent(tx.Querier, node.NodeID, nil)
		if err != nil {
			return err
		}

		a, err := inventory.ToInventoryAgent(tx.Querier, pmmAgent, s.registry)
		if err != nil {
			return err
		}
		res.PmmAgent = a.(*inventorypb.PMMAgent)

		_, err = models.AgentAddNodeExporter(tx.Querier, pmmAgent.AgentID, nil)
		return err
	}); e != nil {
		return nil, e
	}

	return res, nil
}
