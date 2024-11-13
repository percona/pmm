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

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

const (
	defaultTablestatsGroupTableLimit = 1000
	defaultMaxSlowlogFileSize        = 1 << 30 // 1 GB
)

// MySQLService MySQL Management Service.
type MySQLService struct {
	db    *reform.DB
	state agentsStateUpdater
	cc    connectionChecker
	vc    versionCache
	sib   serviceInfoBroker
}

// NewMySQLService creates new MySQL Management Service.
func NewMySQLService(db *reform.DB, state agentsStateUpdater, cc connectionChecker, sib serviceInfoBroker, vc versionCache) *MySQLService {
	return &MySQLService{
		db:    db,
		state: state,
		cc:    cc,
		sib:   sib,
		vc:    vc,
	}
}

// Add adds "MySQL Service", "MySQL Exporter Agent" and "QAN MySQL PerfSchema Agent".
func (s *MySQLService) Add(ctx context.Context, req *managementpb.AddMySQLRequest) (*managementpb.AddMySQLResponse, error) {
	res := &managementpb.AddMySQLResponse{}

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		// tweak according to API docs
		tablestatsGroupTableLimit := req.TablestatsGroupTableLimit
		if tablestatsGroupTableLimit == 0 {
			tablestatsGroupTableLimit = defaultTablestatsGroupTableLimit
		}
		if tablestatsGroupTableLimit < 0 {
			tablestatsGroupTableLimit = -1
		}

		// tweak according to API docs
		maxSlowlogFileSize := req.MaxSlowlogFileSize
		if maxSlowlogFileSize == 0 {
			maxSlowlogFileSize = defaultMaxSlowlogFileSize
		}
		if maxSlowlogFileSize < 0 {
			maxSlowlogFileSize = 0
		}

		nodeID, err := nodeID(tx, req.NodeId, req.NodeName, req.AddNode, req.Address)
		if err != nil {
			return err
		}

		service, err := models.AddNewService(tx.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
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
		res.Service = invService.(*inventorypb.MySQLService) //nolint:forcetypeassert

		req.MetricsMode, err = supportedMetricsMode(tx.Querier, req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		row, err := models.CreateAgent(tx.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
			PMMAgentID:                     req.PmmAgentId,
			ServiceID:                      service.ServiceID,
			Username:                       req.Username,
			Password:                       req.Password,
			AgentPassword:                  req.AgentPassword,
			TLS:                            req.Tls,
			TLSSkipVerify:                  req.TlsSkipVerify,
			MySQLOptions:                   models.MySQLOptionsFromRequest(req),
			TableCountTablestatsGroupLimit: tablestatsGroupTableLimit,
			PushMetrics:                    isPushMode(req.MetricsMode),
			ExposeExporter:                 req.ExposeExporter,
			DisableCollectors:              req.DisableCollectors,
			LogLevel:                       services.SpecifyLogLevel(req.LogLevel, inventorypb.LogLevel_error),
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
			// GetInfoFromService updates the table count in row so, let's also update the response
			res.TableCount = *row.TableCount
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res.MysqldExporter = agent.(*inventorypb.MySQLdExporter) //nolint:forcetypeassert

		if req.QanMysqlPerfschema {
			row, err = models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, &models.CreateAgentParams{
				PMMAgentID:              req.PmmAgentId,
				ServiceID:               service.ServiceID,
				Username:                req.Username,
				Password:                req.Password,
				TLS:                     req.Tls,
				TLSSkipVerify:           req.TlsSkipVerify,
				MySQLOptions:            models.MySQLOptionsFromRequest(req),
				MaxQueryLength:          req.MaxQueryLength,
				QueryExamplesDisabled:   req.DisableQueryExamples,
				CommentsParsingDisabled: req.DisableCommentsParsing,
				LogLevel:                services.SpecifyLogLevel(req.LogLevel, inventorypb.LogLevel_fatal),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanMysqlPerfschema = agent.(*inventorypb.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
		}

		if req.QanMysqlSlowlog {
			row, err = models.CreateAgent(tx.Querier, models.QANMySQLSlowlogAgentType, &models.CreateAgentParams{
				PMMAgentID:              req.PmmAgentId,
				ServiceID:               service.ServiceID,
				Username:                req.Username,
				Password:                req.Password,
				TLS:                     req.Tls,
				TLSSkipVerify:           req.TlsSkipVerify,
				MySQLOptions:            models.MySQLOptionsFromRequest(req),
				MaxQueryLength:          req.MaxQueryLength,
				QueryExamplesDisabled:   req.DisableQueryExamples,
				CommentsParsingDisabled: req.DisableCommentsParsing,
				MaxQueryLogSize:         maxSlowlogFileSize,
				LogLevel:                services.SpecifyLogLevel(req.LogLevel, inventorypb.LogLevel_fatal),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res.QanMysqlSlowlog = agent.(*inventorypb.QANMySQLSlowlogAgent) //nolint:forcetypeassert
		}

		return nil
	}); e != nil {
		return nil, e
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	s.vc.RequestSoftwareVersionsUpdate()

	return res, nil
}
