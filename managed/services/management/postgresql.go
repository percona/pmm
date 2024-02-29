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

const (
	defaultAutoDiscoveryDatabaseLimit = 10
)

// AddPostgreSQL adds "PostgreSQL Service", "PostgreSQL Exporter Agent" and "QAN PostgreSQL PerfSchema Agent".
func (s *ManagementService) AddPostgreSQL(ctx context.Context, req *managementv1.AddPostgreSQLRequest) (*managementv1.AddPostgreSQLResponse, error) {
	res := &managementv1.AddPostgreSQLResponse{}

	if e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		switch {
		case req.AutoDiscoveryLimit == 0:
			req.AutoDiscoveryLimit = defaultAutoDiscoveryDatabaseLimit
		case req.AutoDiscoveryLimit < -1:
			req.AutoDiscoveryLimit = -1
		}

		nodeID, err := nodeID(tx, req.NodeId, req.NodeName, req.AddNode, req.Address)
		if err != nil {
			return err
		}

		service, err := models.AddNewService(tx.Querier, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         nodeID,
			Database:       req.Database,
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
		res.Service = invService.(*inventoryv1.PostgreSQLService) //nolint:forcetypeassert

		req.MetricsMode, err = supportedMetricsMode(tx.Querier, req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		options := models.PostgreSQLOptionsFromRequest(req)
		row, err := models.CreateAgent(tx.Querier, models.PostgresExporterType, &models.CreateAgentParams{
			PMMAgentID:        req.PmmAgentId,
			ServiceID:         service.ServiceID,
			Username:          req.Username,
			Password:          req.Password,
			AgentPassword:     req.AgentPassword,
			TLS:               req.Tls,
			TLSSkipVerify:     req.TlsSkipVerify,
			PushMetrics:       isPushMode(req.MetricsMode),
			ExposeExporter:    req.ExposeExporter,
			DisableCollectors: req.DisableCollectors,
			PostgreSQLOptions: options,
			LogLevel:          services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
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

			// In case of not available PGSM extension it is switch to PGSS.
			if req.QanPostgresqlPgstatmonitorAgent && row.PostgreSQLOptions.PGSMVersion == "" {
				res.Warning = "Could not to detect the pg_stat_monitor extension on your system. Falling back to the pg_stat_statements."
				req.QanPostgresqlPgstatementsAgent = true
				req.QanPostgresqlPgstatmonitorAgent = false
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res.PostgresExporter = agent.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert

		if req.QanPostgresqlPgstatementsAgent {
			row, err = models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatementsAgentType, &models.CreateAgentParams{
				PMMAgentID:              req.PmmAgentId,
				ServiceID:               service.ServiceID,
				Username:                req.Username,
				Password:                req.Password,
				MaxQueryLength:          req.MaxQueryLength,
				QueryExamplesDisabled:   req.DisableQueryExamples,
				CommentsParsingDisabled: req.DisableCommentsParsing,
				TLS:                     req.Tls,
				TLSSkipVerify:           req.TlsSkipVerify,
				PostgreSQLOptions:       options,
				LogLevel:                services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanPostgresqlPgstatementsAgent = agent.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
		}

		if req.QanPostgresqlPgstatmonitorAgent {
			row, err = models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatMonitorAgentType, &models.CreateAgentParams{
				PMMAgentID:              req.PmmAgentId,
				ServiceID:               service.ServiceID,
				Username:                req.Username,
				Password:                req.Password,
				MaxQueryLength:          req.MaxQueryLength,
				QueryExamplesDisabled:   req.DisableQueryExamples,
				CommentsParsingDisabled: req.DisableCommentsParsing,
				TLS:                     req.Tls,
				TLSSkipVerify:           req.TlsSkipVerify,
				PostgreSQLOptions:       options,
				LogLevel:                services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanPostgresqlPgstatmonitorAgent = agent.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent) //nolint:forcetypeassert
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	return res, nil
}
