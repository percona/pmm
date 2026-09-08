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

package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/utils/logger"
)

// NodesService works with inventory API Nodes.
type NodesService struct {
	db            *reform.DB
	r             agentsRegistry
	state         agentsStateUpdater
	vmdb          prometheusService
	grafanaClient grafanaClient
}

// NewNodesService returns Inventory API handler for managing Nodes.
func NewNodesService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService, gc grafanaClient) *NodesService {
	return &NodesService{
		db:            db,
		r:             r,
		state:         state,
		vmdb:          vmdb,
		grafanaClient: gc,
	}
}

// List returns a list of all Nodes.
func (s *NodesService) List(ctx context.Context, filters models.NodeFilters) ([]inventoryv1.Node, error) {
	var nodes []*models.Node
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		nodes, err = models.FindNodes(tx.Querier, filters)
		return err
	})
	if e != nil {
		return nil, e
	}

	res := make([]inventoryv1.Node, len(nodes))
	for i, n := range nodes {
		res[i], e = services.ToAPINode(n)
		if e != nil {
			return nil, e
		}
	}
	return res, nil
}

// Get returns a single Node by ID.
func (s *NodesService) Get(ctx context.Context, req *inventoryv1.GetNodeRequest) (inventoryv1.Node, error) { //nolint:ireturn
	modelNode := &models.Node{}
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

// AddNode adds any type of Node.
func (s *NodesService) AddNode(ctx context.Context, req *inventoryv1.AddNodeRequest) (*inventoryv1.AddNodeResponse, error) {
	res := &inventoryv1.AddNodeResponse{}

	switch req.Node.(type) {
	case *inventoryv1.AddNodeRequest_Generic:
		node, err := s.addGenericNode(ctx, req.GetGeneric())
		if err != nil {
			return nil, err
		}
		res.Node = &inventoryv1.AddNodeResponse_Generic{Generic: node}
	case *inventoryv1.AddNodeRequest_Container:
		node, err := s.addContainerNode(ctx, req.GetContainer())
		if err != nil {
			return nil, err
		}
		res.Node = &inventoryv1.AddNodeResponse_Container{Container: node}
	case *inventoryv1.AddNodeRequest_Remote:
		node, err := s.addRemoteNode(ctx, req.GetRemote())
		if err != nil {
			return nil, err
		}
		res.Node = &inventoryv1.AddNodeResponse_Remote{Remote: node}
	case *inventoryv1.AddNodeRequest_RemoteRds:
		node, err := s.AddRemoteRDSNode(ctx, req.GetRemoteRds())
		if err != nil {
			return nil, err
		}
		res.Node = &inventoryv1.AddNodeResponse_RemoteRds{RemoteRds: node}
	case *inventoryv1.AddNodeRequest_RemoteAzure:
		node, err := s.AddRemoteAzureDatabaseNode(ctx, req.GetRemoteAzure())
		if err != nil {
			return nil, err
		}
		res.Node = &inventoryv1.AddNodeResponse_RemoteAzureDatabase{RemoteAzureDatabase: node}
	default:
		return nil, fmt.Errorf("invalid request %v", req.GetNode())
	}

	return res, nil
}

// AddGenericNode adds Generic Node.
func (s *NodesService) addGenericNode(ctx context.Context, req *inventoryv1.AddGenericNodeParams) (*inventoryv1.GenericNode, error) {
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
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

	return invNode.(*inventoryv1.GenericNode), nil //nolint:forcetypeassert
}

// AddContainerNode adds Container Node.
func (s *NodesService) addContainerNode(ctx context.Context, req *inventoryv1.AddContainerNodeParams) (*inventoryv1.ContainerNode, error) {
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
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

	return invNode.(*inventoryv1.ContainerNode), nil //nolint:forcetypeassert
}

// AddRemoteNode adds Remote Node.
func (s *NodesService) addRemoteNode(ctx context.Context, req *inventoryv1.AddRemoteNodeParams) (*inventoryv1.RemoteNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

	return invNode.(*inventoryv1.RemoteNode), nil //nolint:forcetypeassert
}

// AddRemoteRDSNode adds a new RDS node.
func (s *NodesService) AddRemoteRDSNode(ctx context.Context, req *inventoryv1.AddRemoteRDSNodeParams) (*inventoryv1.RemoteRDSNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

	return invNode.(*inventoryv1.RemoteRDSNode), nil //nolint:forcetypeassert
}

// AddRemoteAzureDatabaseNode adds a new Azure database node
//
//nolint:dupl
func (s *NodesService) AddRemoteAzureDatabaseNode(ctx context.Context, req *inventoryv1.AddRemoteAzureNodeParams) (*inventoryv1.RemoteAzureDatabaseNode, error) {
	params := &models.CreateNodeParams{
		NodeName:     req.NodeName,
		Address:      req.Address,
		NodeModel:    req.NodeModel,
		Region:       pointer.ToStringOrNil(req.Region),
		AZ:           req.Az,
		CustomLabels: req.CustomLabels,
	}

	node := &models.Node{}
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

	return invNode.(*inventoryv1.RemoteAzureDatabaseNode), nil //nolint:forcetypeassert
}

// removeServiceAccount deletes the Grafana service account of a Node being removed. Pmm-agent
// authenticates with a token of the account named after the Node, so the account goes with the Node.
//
// Force is deliberately not taken: removing a Node means "the Node with everything on it", while
// DeleteServiceAccount reads its own force as "delete the account even when it holds tokens nobody here
// created", which removing a Node does not ask for. Grafana keeps such an account, deletes only
// pmm-agent's own token, and says so in the warning returned here.
func (s *NodesService) removeServiceAccount(ctx context.Context, nodeName string) (string, error) {
	warning, err := services.RemoveNodeServiceAccount(ctx, s.grafanaClient, nodeName, false)
	switch {
	case errors.Is(err, services.ErrServiceAccountNotFound):
		// A Node no pmm-agent ever registered, a remote or an RDS one among them, has no account.
		logger.Get(ctx).Debugf("Node %s had no service account to delete.", nodeName)
		return "", nil
	case err != nil:
		return "", status.Errorf(codes.Unavailable, "Node %s was not removed: its Grafana service account"+
			" could not be deleted, and removing the Node would leave its token behind. %s", nodeName, err)
	case warning != "":
		// Also on the record here: the response reaches one caller, who may discard it, while a credential
		// outliving its Node is worth being able to find afterwards.
		logger.Get(ctx).Warnf("Service account of node %s: %s", nodeName, warning)
	}

	return warning, nil
}

// agentsToNotify names the pmm-agents which have to hear about a Node removal: those running on the
// Node, which go with it, and those whose state changes because something they monitor did.
type agentsToNotify struct {
	kick     map[string]struct{}
	setState map[string]struct{}
}

// agentsAffectedBy collects the pmm-agents a cascading removal of the given Node reaches.
func agentsAffectedBy(q *reform.Querier, nodeID string) (agentsToNotify, error) {
	notify := agentsToNotify{
		kick:     make(map[string]struct{}),
		setState: make(map[string]struct{}),
	}

	agents, err := models.FindPMMAgentsRunningOnNode(q, nodeID)
	if err != nil {
		return notify, fmt.Errorf("failed to get pmm-agents running on node %s: %w", nodeID, err)
	}
	for _, a := range agents {
		notify.kick[a.AgentID] = struct{}{}
	}

	agents, err = models.FindAgents(q, models.AgentFilters{NodeID: nodeID})
	if err != nil {
		return notify, fmt.Errorf("failed to get agents on node %s: %w", nodeID, err)
	}
	for _, a := range agents {
		if a.PMMAgentID != nil {
			notify.setState[pointer.GetString(a.PMMAgentID)] = struct{}{}
		}
	}

	agents, err = models.FindPMMAgentsForServicesOnNode(q, nodeID)
	if err != nil {
		return notify, fmt.Errorf("failed to get pmm-agents for services on node %s: %w", nodeID, err)
	}
	for _, a := range agents {
		notify.setState[a.AgentID] = struct{}{}
	}

	return notify, nil
}

// Remove removes Node without any Agents and Services.
// Removes Node with the Agents and Services if force == true.
// Returns an error if force == false and Node has Agents or Services.
// The returned warning names what the removal left behind on purpose, which is the Grafana service
// account when it holds tokens pmm-agent did not create.
func (s *NodesService) Remove(ctx context.Context, id string, force bool) (string, error) {
	node, err := models.FindNodeByID(s.db.Querier, id)
	if err != nil {
		return "", err
	}

	var warning string
	var notify agentsToNotify

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if force {
			mode = models.RemoveCascade

			var err error
			notify, err = agentsAffectedBy(tx.Querier, id)
			if err != nil {
				return err
			}
		}
		err := models.RemoveNode(tx.Querier, id, mode)
		if err != nil {
			return err
		}

		// Inside the transaction, so that a Grafana which cannot be reached takes the removal down with it:
		// removing the Node while its account survives leaves a live Admin credential for a host which no
		// longer exists, and by then nothing is in a position to put either back. Refusing keeps the two in
		// step, and leaves the operator with a Node they can remove again once Grafana is up.
		warning, err = s.removeServiceAccount(ctx, node.NodeName)

		return err
	})
	if e != nil {
		return "", e
	}

	for id := range notify.setState {
		s.state.RequestStateUpdate(ctx, id)
	}
	for id := range notify.kick {
		s.r.Kick(ctx, id)
	}

	if force {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		s.vmdb.RequestConfigurationUpdate()
	}

	return warning, nil
}
