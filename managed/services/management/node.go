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
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	nodev1beta1 "github.com/percona/pmm/api/managementpb/node"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
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
func (s *MgmtNodeService) ListNodes(ctx context.Context, req *nodev1beta1.ListNodeRequest) (*nodev1beta1.ListNodeResponse, error) {
	filters := models.NodeFilters{
		NodeType: services.ProtoToModelNodeType(req.NodeType),
	}

	var nodes []*models.Node
	var agents []*models.Agent
	var services []*models.Service

	errTX := s.db.InTransactionContext(s.db.Querier.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
		var err error

		nodes, err = models.FindNodes(s.db.Querier, filters)
		if err != nil {
			return err
		}

		agents, err = models.FindAgents(s.db.Querier, models.AgentFilters{})
		if err != nil {
			return err
		}

		services, err = models.FindServices(s.db.Querier, models.ServiceFilters{})
		if err != nil {
			return err
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	convertAgentToProto := func(agent *models.Agent) *nodev1beta1.UniversalNode_Agent {
		return &nodev1beta1.UniversalNode_Agent{
			AgentId:     agent.AgentID,
			AgentType:   string(agent.AgentType),
			Status:      agent.Status,
			IsConnected: s.r.IsConnected(agent.AgentID),
		}
	}

	aMap := make(map[string][]*nodev1beta1.UniversalNode_Agent)
	for _, a := range agents {
		if a.NodeID != nil || a.RunsOnNodeID != nil {
			var nodeID string
			if a.NodeID != nil {
				nodeID = pointer.GetString(a.NodeID)
			} else {
				nodeID = pointer.GetString(a.RunsOnNodeID)
			}
			aMap[nodeID] = append(aMap[nodeID], convertAgentToProto(a))
		}
	}

	sMap := make(map[string][]*nodev1beta1.UniversalNode_Service, len(services))
	for _, s := range services {
		sMap[s.NodeID] = append(sMap[s.NodeID], &nodev1beta1.UniversalNode_Service{
			ServiceId:   s.ServiceID,
			ServiceType: string(s.ServiceType),
			ServiceName: s.ServiceName,
		})
	}

	// NOTE: this query will need to be updated if we start supporting more exporters (ex: gcp_exporter).
	query := `up{job=~".*_hr$",agent_type=~"node_exporter|rds_exporter|external-exporter"}`

	result, _, err := s.vmClient.Query(ctx, query, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute an instant VM query")
	}

	metrics := make(map[string]int, len(result.(model.Vector)))
	for _, v := range result.(model.Vector) { //nolint:forcetypeassert
		nodeID := string(v.Metric[model.LabelName("node_id")])
		// Sometimes we may see several metrics for the same node, so we just take the first one.
		if _, ok := metrics[nodeID]; !ok {
			metrics[nodeID] = int(v.Value)
		}
	}

	res := make([]*nodev1beta1.UniversalNode, len(nodes))
	for i, node := range nodes {
		labels, err := node.GetCustomLabels()
		if err != nil {
			return nil, err
		}

		uNode := &nodev1beta1.UniversalNode{
			Address:      node.Address,
			CustomLabels: labels,
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			NodeType:     string(node.NodeType),
		}

		if metric, ok := metrics[node.NodeID]; ok {
			switch metric {
			// We assume there can only be metric values of either 1(UP) or 0(DOWN).
			case 0:
				uNode.Status = nodev1beta1.UniversalNode_DOWN
			case 1:
				uNode.Status = nodev1beta1.UniversalNode_UP
			}
		} else {
			uNode.Status = nodev1beta1.UniversalNode_UNKNOWN
		}

		if uAgents, ok := aMap[node.NodeID]; ok {
			uNode.Agents = uAgents
		}

		if uServices, ok := sMap[node.NodeID]; ok {
			uNode.Services = uServices
		}

		res[i] = uNode
	}

	return &nodev1beta1.ListNodeResponse{
		Nodes: res,
	}, nil
}

// GetNode returns a single Node by ID.
//
//nolint:unparam
func (s *MgmtNodeService) GetNode(ctx context.Context, req *nodev1beta1.GetNodeRequest) (*nodev1beta1.GetNodeResponse, error) {
	node, err := models.FindNodeByID(s.db.Querier, req.NodeId)
	if err != nil {
		return nil, err
	}

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

	return &nodev1beta1.GetNodeResponse{
		Node: uNode,
	}, nil
}
