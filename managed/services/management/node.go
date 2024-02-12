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
	"math/rand"
	"net/http"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// NodeService represents service for working with nodes.
type NodeService struct {
	db  *reform.DB
	akp apiKeyProvider

	l *logrus.Entry

	managementv1.UnimplementedNodeServiceServer
}

// NewNodeService creates NodeService instance.
func NewNodeService(db *reform.DB, akp apiKeyProvider) *NodeService {
	return &NodeService{
		db:  db,
		akp: akp,
		l:   logrus.WithField("component", "node"),
	}
}

// RegisterNode performs the registration of a new node.
func (s *NodeService) RegisterNode(ctx context.Context, req *managementv1.RegisterNodeRequest) (*managementv1.RegisterNodeResponse, error) {
	res := &managementv1.RegisterNodeResponse{}

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

	// get authorization from headers.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		msg := "Couldn't create Admin API Key: cannot get headers from metadata"
		s.l.Errorln(msg)
		res.Warning = msg
		return res, nil
	}
	authorizationHeaders := md.Get("Authorization")
	if len(authorizationHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Authorization error.")
	}
	headers := make(http.Header)
	headers.Set("Authorization", authorizationHeaders[0])
	if !s.akp.IsAPIKeyAuth(headers) {
		apiKeyName := fmt.Sprintf("pmm-agent-%s-%d", req.NodeName, rand.Int63()) //nolint:gosec
		_, res.Token, e = s.akp.CreateAdminAPIKey(ctx, apiKeyName)
		if e != nil {
			msg := fmt.Sprintf("Couldn't create Admin API Key: %s", e)
			s.l.Errorln(msg)
			res.Warning = msg
		}
	}

	return res, nil
}
