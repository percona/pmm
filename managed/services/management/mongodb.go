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
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// AddMongoDB adds "MongoDB Service", "MongoDB Exporter Agent" and "QAN MongoDB Profiler".
func (s *ManagementService) addMongoDB(ctx context.Context, req *managementv1.AddMongoDBServiceParams) (*managementv1.AddServiceResponse, error) {
	mongodb := &managementv1.MongoDBServiceResult{}

	if e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		nodeID, err := nodeID(tx, req.NodeId, req.NodeName, req.AddNode, req.Address)
		if err != nil {
			return err
		}
		service, err := models.AddNewService(tx.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         nodeID,
			Environment:    req.Environment,
			Cluster:        req.Cluster,
			ReplicationSet: req.ReplicationSet,
			Address:        pointer.ToStringOrNil(req.Address),
			Port:           pointer.ToUint16OrNil(uint16(req.Port)),
			Socket:         pointer.ToStringOrNil(req.Socket),
			CustomLabels:   req.CustomLabels,
		})
		if err != nil {
			return err
		}

		invService, err := services.ToAPIService(service)
		if err != nil {
			return err
		}
		mongodb.Service = invService.(*inventoryv1.MongoDBService) //nolint:forcetypeassert

		req.MetricsMode, err = supportedMetricsMode(tx.Querier, req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		row, err := models.CreateAgent(tx.Querier, models.MongoDBExporterType, &models.CreateAgentParams{
			PMMAgentID:     req.PmmAgentId,
			ServiceID:      service.ServiceID,
			Username:       req.Username,
			Password:       req.Password,
			AgentPassword:  req.AgentPassword,
			TLS:            req.Tls,
			TLSSkipVerify:  req.TlsSkipVerify,
			MongoDBOptions: models.MongoDBOptionsFromRequest(req),
			LogLevel:       services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			ExporterOptions: &models.ExporterOptions{
				ExposeExporter:     req.ExposeExporter,
				PushMetrics:        isPushMode(req.MetricsMode),
				DisabledCollectors: req.DisableCollectors,
			},
		})
		if err != nil {
			return err
		}

		if !req.SkipConnectionCheck {
			if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}

			if err = s.sib.GetInfoFromService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		mongodb.MongodbExporter = agent.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert

		if req.QanMongodbProfiler {
			row, err = models.CreateAgent(tx.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
				PMMAgentID:    req.PmmAgentId,
				ServiceID:     service.ServiceID,
				Username:      req.Username,
				Password:      req.Password,
				TLS:           req.Tls,
				TLSSkipVerify: req.TlsSkipVerify,
				QANOptions: &models.QANOptions{
					MaxQueryLength: req.MaxQueryLength,
					// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-7860
				},
				MongoDBOptions: models.MongoDBOptionsFromRequest(req),
				LogLevel:       services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			})
			if err != nil {
				return err
			}

			agent, err := services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			mongodb.QanMongodbProfiler = agent.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	s.vc.RequestSoftwareVersionsUpdate()

	res := &managementv1.AddServiceResponse{
		Service: &managementv1.AddServiceResponse_Mongodb{
			Mongodb: mongodb,
		},
	}

	return res, nil
}
