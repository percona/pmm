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

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/auth"
)

// RegisterNode performs the registration of a new node.
func (s *ManagementService) RegisterNode(ctx context.Context, req *managementv1.RegisterNodeRequest) (*managementv1.RegisterNodeResponse, error) {
	res := &managementv1.RegisterNodeResponse{}

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
		case *inventoryv1.GenericNode:
			res.GenericNode = n
		case *inventoryv1.ContainerNode:
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
		res.PmmAgent = a.(*inventoryv1.PMMAgent) //nolint:forcetypeassert
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
		_, res.Token, e = s.grafanaClient.CreateServiceAccount(ctx, req.NodeName, req.Reregister)
		if e != nil {
			return nil, e
		}
	}

	return res, nil
}

// Unregister do unregistration of the node.
func (s *ManagementService) Unregister(ctx context.Context, req *managementv1.UnregisterNodeRequest) (*managementv1.UnregisterNodeResponse, error) {
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

	warning, err := s.grafanaClient.DeleteServiceAccount(ctx, node.NodeName, req.Force)
	if err != nil {
		// TODO: need to pass the logger to the service
		// s.l.WithError(err).Error("deleting service account")
		return &managementv1.UnregisterNodeResponse{ //nolint:nilerr
			Warning: err.Error(),
		}, nil
	}

	return &managementv1.UnregisterNodeResponse{
		Warning: warning,
	}, nil
}
