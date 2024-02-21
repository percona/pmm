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
func (s *ManagementService) AddMongoDB(ctx context.Context, req *managementv1.AddMongoDBRequest) (*managementv1.AddMongoDBResponse, error) {
	res := &managementv1.AddMongoDBResponse{}

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
		res.Service = invService.(*inventoryv1.MongoDBService) //nolint:forcetypeassert

		mongoDBOptions := models.MongoDBOptionsFromRequest(req)

		req.MetricsMode, err = supportedMetricsMode(tx.Querier, req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		row, err := models.CreateAgent(tx.Querier, models.MongoDBExporterType, &models.CreateAgentParams{
			PMMAgentID:        req.PmmAgentId,
			ServiceID:         service.ServiceID,
			Username:          req.Username,
			Password:          req.Password,
			AgentPassword:     req.AgentPassword,
			TLS:               req.Tls,
			TLSSkipVerify:     req.TlsSkipVerify,
			MongoDBOptions:    mongoDBOptions,
			PushMetrics:       isPushMode(req.MetricsMode),
			ExposeExporter:    req.ExposeExporter,
			DisableCollectors: req.DisableCollectors,
			LogLevel:          services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
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
		res.MongodbExporter = agent.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert

		if req.QanMongodbProfiler {
			row, err = models.CreateAgent(tx.Querier, models.QANMongoDBProfilerAgentType, &models.CreateAgentParams{
				PMMAgentID:     req.PmmAgentId,
				ServiceID:      service.ServiceID,
				Username:       req.Username,
				Password:       req.Password,
				TLS:            req.Tls,
				TLSSkipVerify:  req.TlsSkipVerify,
				MongoDBOptions: mongoDBOptions,
				MaxQueryLength: req.MaxQueryLength,
				LogLevel:       services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
				// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-7860
			})
			if err != nil {
				return err
			}

			agent, err := services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanMongodbProfiler = agent.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	s.vc.RequestSoftwareVersionsUpdate()

	return res, nil
}
