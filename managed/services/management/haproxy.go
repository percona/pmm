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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// HAProxyService HAProxy Management Service.
type HAProxyService struct {
	db    *reform.DB
	vmdb  prometheusService
	state agentsStateUpdater
	cc    connectionChecker

	managementpb.UnimplementedHAProxyServer
}

// NewHAProxyService creates new HAProxy Management Service.
func NewHAProxyService(db *reform.DB, vmdb prometheusService, state agentsStateUpdater, cc connectionChecker) *HAProxyService {
	return &HAProxyService{
		db:    db,
		vmdb:  vmdb,
		state: state,
		cc:    cc,
	}
}

// AddHAProxy adds an HAProxy service based on the provided request.
func (e HAProxyService) AddHAProxy(ctx context.Context, req *managementpb.AddHAProxyRequest) (*managementpb.AddHAProxyResponse, error) {
	res := &managementpb.AddHAProxyResponse{}
	var pmmAgentID *string
	if e := e.db.InTransaction(func(tx *reform.TX) error {
		if req.Address == "" && req.AddNode != nil {
			return status.Error(codes.InvalidArgument, "address can't be empty for add node request.")
		}
		nodeID, err := nodeID(tx, req.NodeId, req.NodeName, req.AddNode, req.Address)
		if err != nil {
			return err
		}

		service, err := models.AddNewService(tx.Querier, models.HAProxyServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         nodeID,
			Environment:    req.Environment,
			Cluster:        req.Cluster,
			ReplicationSet: req.ReplicationSet,
			CustomLabels:   req.CustomLabels,
		})
		if err != nil {
			return err
		}

		invService, err := services.ToAPIService(service)
		if err != nil {
			return err
		}
		res.Service = invService.(*inventorypb.HAProxyService) //nolint:forcetypeassert

		if req.MetricsMode == managementpb.MetricsMode_AUTO {
			agentIDs, err := models.FindPMMAgentsRunningOnNode(tx.Querier, req.NodeId)
			switch {
			case err != nil || len(agentIDs) != 1:
				req.MetricsMode = managementpb.MetricsMode_PULL
			default:
				req.MetricsMode, err = supportedMetricsMode(tx.Querier, req.MetricsMode, agentIDs[0].AgentID)
				if err != nil {
					return err
				}
			}
		}

		params := &models.CreateExternalExporterParams{
			RunsOnNodeID: nodeID,
			ServiceID:    service.ServiceID,
			Username:     req.Username,
			Password:     req.Password,
			Scheme:       req.Scheme,
			MetricsPath:  req.MetricsPath,
			ListenPort:   req.ListenPort,
			CustomLabels: req.CustomLabels,
			PushMetrics:  isPushMode(req.MetricsMode),
		}
		row, err := models.CreateExternalExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		if !req.SkipConnectionCheck {
			if err = e.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res.ExternalExporter = agent.(*inventorypb.ExternalExporter) //nolint:forcetypeassert
		pmmAgentID = row.PMMAgentID

		return nil
	}); e != nil {
		return nil, e
	}
	// we have to trigger after transaction
	if pmmAgentID != nil {
		// It's required to regenerate victoriametrics config file.
		e.state.RequestStateUpdate(ctx, *pmmAgentID)
	} else {
		e.vmdb.RequestConfigurationUpdate()
	}
	return res, nil
}
