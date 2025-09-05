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

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// AddHAProxy adds an HAProxy service based on the provided request.
func (s *ManagementService) addHAProxy(ctx context.Context, req *managementv1.AddHAProxyServiceParams) (*managementv1.AddServiceResponse, error) {
	var pmmAgentID *string
	haproxy := &managementv1.HAProxyServiceResult{}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
		haproxy.Service = invService.(*inventoryv1.HAProxyService) //nolint:forcetypeassert

		if req.MetricsMode == managementv1.MetricsMode_METRICS_MODE_UNSPECIFIED {
			agentIDs, err := models.FindPMMAgentsRunningOnNode(tx.Querier, req.NodeId)
			switch {
			case err != nil || len(agentIDs) != 1:
				req.MetricsMode = managementv1.MetricsMode_METRICS_MODE_PULL
			default:
				req.MetricsMode, err = supportedMetricsMode(req.MetricsMode, agentIDs[0].AgentID)
				if err != nil {
					return err
				}
			}
		}

		params := &models.CreateExternalExporterParams{
			RunsOnNodeID:  nodeID,
			ServiceID:     service.ServiceID,
			Username:      req.Username,
			Password:      req.Password,
			Scheme:        req.Scheme,
			MetricsPath:   req.MetricsPath,
			ListenPort:    req.ListenPort,
			CustomLabels:  req.CustomLabels,
			PushMetrics:   isPushMode(req.MetricsMode),
			TLSSkipVerify: req.TlsSkipVerify,
		}
		row, err := models.CreateExternalExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		if !req.SkipConnectionCheck {
			if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		haproxy.ExternalExporter = agent.(*inventoryv1.ExternalExporter) //nolint:forcetypeassert
		pmmAgentID = row.PMMAgentID

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}
	// we have to trigger updates once the transaction completes
	if pmmAgentID != nil {
		// It's required to regenerate victoriametrics config file.
		s.state.RequestStateUpdate(ctx, *pmmAgentID)
	} else {
		s.vmdb.RequestConfigurationUpdate()
	}

	res := &managementv1.AddServiceResponse{
		Service: &managementv1.AddServiceResponse_Haproxy{
			Haproxy: haproxy,
		},
	}

	return res, nil
}
