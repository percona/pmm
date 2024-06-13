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

package management

import (
	"context"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/auth"
)

// NodeService represents service for working with nodes.
type NodeService struct {
	db    *reform.DB
	ap    authProvider
	l     *logrus.Entry
	r     agentsRegistry
	state agentsStateUpdater
	vmdb  prometheusService
}

// NewNodeService creates NodeService instance.
func NewNodeService(db *reform.DB, ap authProvider, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService) *NodeService {
	return &NodeService{
		db:    db,
		ap:    ap,
		r:     r,
		state: state,
		vmdb:  vmdb,
		l:     logrus.WithField("component", "node"),
	}
}

// Register do registration of the new node.
func (s *NodeService) Register(ctx context.Context, req *managementpb.RegisterNodeRequest) (*managementpb.RegisterNodeResponse, error) {
	res := &managementpb.RegisterNodeResponse{}

	fmt.Printf("\n\n %t \n\n", req.Reregister)
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
		res.PmmAgent = a.(*inventorypb.PMMAgent) //nolint:forcetypeassert
		_, err = models.
			CreateNodeExporter(tx.Querier, pmmAgent.AgentID, nil, isPushMode(req.MetricsMode), req.ExposeExporter,
				req.DisableCollectors, pointer.ToStringOrNil(req.AgentPassword), "")
		return err
	})
	if e != nil {
		return nil, e
	}

	authHeaders, _ := auth.GetHeadersFromContext(ctx)
	token := auth.GetTokenFromHeaders(authHeaders)
	if token != "" {
		res.Token = token
	} else {
		_, res.Token, e = s.ap.CreateServiceAccount(ctx, req.NodeName, req.Reregister)
		if e != nil {
			return nil, e
		}
	}

	return res, nil
}

// Unregister do unregistration of the node.
func (s *NodeService) Unregister(ctx context.Context, req *managementpb.UnregisterNodeRequest) (*managementpb.UnregisterNodeResponse, error) {
	idsToKick := make(map[string]struct{})
	idsToSetState := make(map[string]struct{})

	node, err := models.FindNodeByID(s.db.Querier, req.NodeId)
	if err != nil {
		return nil, err
	}

	if e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if req.Force {
			mode = models.RemoveCascade

			agents, err := models.FindPMMAgentsRunningOnNode(tx.Querier, node.NodeID)
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				idsToKick[a.AgentID] = struct{}{}
			}

			agents, err = models.FindAgents(tx.Querier, models.AgentFilters{NodeID: node.NodeID})
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				if a.PMMAgentID != nil {
					idsToSetState[pointer.GetString(a.PMMAgentID)] = struct{}{}
				}
			}

			agents, err = models.FindPMMAgentsForServicesOnNode(tx.Querier, node.NodeID)
			if err != nil {
				return errors.WithStack(err)
			}
			for _, a := range agents {
				idsToSetState[a.AgentID] = struct{}{}
			}
		}
		return models.RemoveNode(tx.Querier, node.NodeID, mode)
	}); e != nil {
		return nil, e
	}

	for id := range idsToSetState {
		s.state.RequestStateUpdate(ctx, id)
	}
	for id := range idsToKick {
		s.r.Kick(ctx, id)
	}

	if req.Force {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		s.vmdb.RequestConfigurationUpdate()
	}

	warning, err := s.ap.DeleteServiceAccount(ctx, node.NodeName, req.Force)
	if err != nil {
		s.l.WithError(err).Error("deleting service account")
		return &managementpb.UnregisterNodeResponse{
			Warning: err.Error(),
		}, nil
	}

	return &managementpb.UnregisterNodeResponse{
		Warning: warning,
	}, nil
}
