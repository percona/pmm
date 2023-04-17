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
	"fmt"
	"math/rand"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	agentv1beta1 "github.com/percona/pmm/api/managementpb/agent"
	nodev1beta1 "github.com/percona/pmm/api/managementpb/node"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/inventory/grpc"
)

//go:generate ../../../bin/mockery -name=apiKeyProvider -case=snake -inpkg -testonly

type apiKeyProvider interface {
	CreateAdminAPIKey(ctx context.Context, name string) (int64, string, error)
}

// NodeService represents service for working with nodes.
type NodeService struct {
	db  *reform.DB
	akp apiKeyProvider
}

// MgmtNodeService represents a management API service for working with nodes.
type MgmtNodeService struct {
	db       *reform.DB
	r        agentsRegistry
	vmClient victoriaMetricsClient

	nodev1beta1.UnimplementedMgmtNodeServer
}

// NewNodeService creates NodeService instance.
func NewNodeService(db *reform.DB, akp apiKeyProvider) *NodeService {
	return &NodeService{
		db:  db,
		akp: akp,
	}
}

// NewMgmtNodeService creates MgmtNodeService instance.
func NewMgmtNodeService(db *reform.DB, r agentsRegistry, vmClient victoriaMetricsClient) *MgmtNodeService {
	return &MgmtNodeService{
		db:       db,
		r:        r,
		vmClient: vmClient,
	}
}

// Register do registration of the new node.
func (s *NodeService) Register(ctx context.Context, req *managementpb.RegisterNodeRequest) (*managementpb.RegisterNodeResponse, error) {
	res := &managementpb.RegisterNodeResponse{}

	e := s.db.InTransaction(func(tx *reform.TX) error {
		node, err := models.FindNodeByName(tx.Querier, req.NodeName)
		switch status.Code(err) { //nolint:exhaustive
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

		node, err = models.CheckUniqueNodeInstanceRegion(tx.Querier, req.Address, &req.Region)
		switch status.Code(err) { //nolint:exhaustive
		case codes.OK:
			// nothing
		case codes.AlreadyExists:
			if !req.Reregister {
				return err
			}
			err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade)
		}
		if err != nil {
			return err
		}

		nodeType, err := nodeType(req.NodeType)
		if err != nil {
			return err
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

		n, err := services.ToAPINode(node)
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

		pmmAgent, err := models.CreatePMMAgent(tx.Querier, node.NodeID, nil)
		if err != nil {
			return err
		}

		a, err := services.ToAPIAgent(tx.Querier, pmmAgent)
		if err != nil {
			return err
		}
		res.PmmAgent = a.(*inventorypb.PMMAgent)
		_, err = models.
			CreateNodeExporter(tx.Querier, pmmAgent.AgentID, nil, isPushMode(req.MetricsMode), req.DisableCollectors,
				pointer.ToStringOrNil(req.AgentPassword), "")
		return err
	})
	if e != nil {
		return nil, e
	}

	apiKeyName := fmt.Sprintf("pmm-agent-%s-%d", req.NodeName, rand.Int63()) //nolint:gosec
	_, res.Token, e = s.akp.CreateAdminAPIKey(ctx, apiKeyName)
	if e != nil {
		return nil, e
	}

	return res, nil
}

// ListNodes returns a filtered list of Nodes.
//
//nolint:unparam
func (s *MgmtNodeService) ListNodes(ctx context.Context, req *nodev1beta1.ListNodeRequest) (*nodev1beta1.ListNodeResponse, error) {
	filters := models.NodeFilters{
		NodeType: grpc.ProtoToModelNodeType(req.NodeType),
	}

	nodes, err := models.FindNodes(s.db.Querier, filters)
	if err != nil {
		return nil, err
	}

	agentToAPI := func(agent *models.Agent) *agentv1beta1.UniversalAgent {
		return &agentv1beta1.UniversalAgent{
			AgentId:     agent.AgentID,
			AgentType:   string(agent.AgentType),
			Status:      agent.Status,
			IsConnected: s.r.IsConnected(agent.AgentID),
		}
	}

	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{})
	if err != nil {
		return nil, err
	}

	res := make([]*nodev1beta1.UniversalNode, len(nodes))
	for i, node := range nodes {
		labels, err := node.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		uNode := &nodev1beta1.UniversalNode{
			Address:       node.Address,
			Az:            node.AZ,
			CreatedAt:     timestamppb.New(node.CreatedAt),
			ContainerId:   pointer.GetString(node.ContainerID),
			ContainerName: pointer.GetString(node.ContainerName),
			CustomLabels:  labels,
			Distro:        node.Distro,
			MachineId:     pointer.GetString(node.MachineID),
			NodeId:        node.NodeID,
			NodeName:      node.NodeName,
			NodeType:      string(node.NodeType),
			NodeModel:     node.NodeModel,
			Region:        pointer.GetString(node.Region),
			UpdatedAt:     timestamppb.New(node.UpdatedAt),
		}

		var svcAgents []*agentv1beta1.UniversalAgent

		for _, agent := range agents {
			if agent.ServiceID == nil {
				svcAgents = append(svcAgents, agentToAPI(agent))
			}
		}

		uNode.Agents = svcAgents
		res[i] = uNode
	}

	return &nodev1beta1.ListNodeResponse{
		Nodes: res,
	}, nil
}
