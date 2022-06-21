// pmm-managed
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

	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// PostgreSQLService PostgreSQL Management Service.
type PostgreSQLService struct {
	db    *reform.DB
	state agentsStateUpdater
	cc    connectionChecker
}

// NewPostgreSQLService creates new PostgreSQL Management Service.
func NewPostgreSQLService(db *reform.DB, state agentsStateUpdater, cc connectionChecker) *PostgreSQLService {
	return &PostgreSQLService{
		db:    db,
		state: state,
		cc:    cc,
	}
}

// Add adds "PostgreSQL Service", "PostgreSQL Exporter Agent" and "QAN PostgreSQL PerfSchema Agent".
func (s *PostgreSQLService) Add(ctx context.Context, req *managementpb.AddPostgreSQLRequest) (*managementpb.AddPostgreSQLResponse, error) {
	res := &managementpb.AddPostgreSQLResponse{}

	if e := s.db.InTransaction(func(tx *reform.TX) error {
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
		res.Service = invService.(*inventorypb.PostgreSQLService)

		req.MetricsMode, err = supportedMetricsMode(tx.Querier, req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		row, err := models.CreateAgent(tx.Querier, models.PostgresExporterType, &models.CreateAgentParams{
			PMMAgentID:        req.PmmAgentId,
			ServiceID:         service.ServiceID,
			Username:          req.Username,
			Password:          req.Password,
			AgentPassword:     req.AgentPassword,
			TLS:               req.Tls,
			TLSSkipVerify:     req.TlsSkipVerify,
			PushMetrics:       isPushMode(req.MetricsMode),
			DisableCollectors: req.DisableCollectors,
			PostgreSQLOptions: models.PostgreSQLOptionsFromRequest(req),
			LogLevel:          specifyLogLevel(req.LogLevel),
		})
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
		res.PostgresExporter = agent.(*inventorypb.PostgresExporter)

		if req.QanPostgresqlPgstatementsAgent {
			row, err = models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatementsAgentType, &models.CreateAgentParams{
				PMMAgentID:        req.PmmAgentId,
				ServiceID:         service.ServiceID,
				Username:          req.Username,
				Password:          req.Password,
				TLS:               req.Tls,
				TLSSkipVerify:     req.TlsSkipVerify,
				PostgreSQLOptions: models.PostgreSQLOptionsFromRequest(req),
				LogLevel:          specifyLogLevel(req.LogLevel),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanPostgresqlPgstatementsAgent = agent.(*inventorypb.QANPostgreSQLPgStatementsAgent)
		}

		if req.QanPostgresqlPgstatmonitorAgent {
			row, err = models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatMonitorAgentType, &models.CreateAgentParams{
				PMMAgentID:            req.PmmAgentId,
				ServiceID:             service.ServiceID,
				Username:              req.Username,
				Password:              req.Password,
				QueryExamplesDisabled: req.DisableQueryExamples,
				TLS:                   req.Tls,
				TLSSkipVerify:         req.TlsSkipVerify,
				PostgreSQLOptions:     models.PostgreSQLOptionsFromRequest(req),
				LogLevel:              specifyLogLevel(req.LogLevel),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanPostgresqlPgstatmonitorAgent = agent.(*inventorypb.QANPostgreSQLPgStatMonitorAgent)
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	return res, nil
}
