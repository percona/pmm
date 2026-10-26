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
	"github.com/percona/pmm/managed/utils/duration"
)

const (
	defaultTablestatsGroupTableLimit = 1000
	defaultMaxSlowlogFileSize        = 1 << 30 // 1 GB
)

// storedTablestatsGroupTableLimit converts the limit as documented in the API to the stored form:
// zero selects the default and any negative value disables the tablestats group.
func storedTablestatsGroupTableLimit(limit int32) int32 {
	switch {
	case limit == 0:
		return defaultTablestatsGroupTableLimit
	case limit < 0:
		return -1
	default:
		return limit
	}
}

// storedMaxSlowlogFileSize converts the size as documented in the API to the stored form:
// zero selects the default and any negative value, stored as zero, disables rotation.
func storedMaxSlowlogFileSize(size int64) int64 {
	switch {
	case size == 0:
		return defaultMaxSlowlogFileSize
	case size < 0:
		return 0
	default:
		return size
	}
}

// AddMySQL adds "MySQL Service", "MySQL Exporter Agent" and "QAN MySQL PerfSchema Agent".
func (s *ManagementService) addMySQL(ctx context.Context, req *managementv1.AddMySQLServiceParams) (*managementv1.AddServiceResponse, error) { //nolint:gocognit
	mysql := &managementv1.MySQLServiceResult{}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		tablestatsGroupTableLimit := storedTablestatsGroupTableLimit(req.TablestatsGroupTableLimit)
		maxSlowlogFileSize := storedMaxSlowlogFileSize(req.MaxSlowlogFileSize)

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
			Port:           pointer.ToUint16OrNil(uint16(req.Port)), //nolint:gosec // port is a uint16
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
		mysql.Service = invService.(*inventoryv1.MySQLService) //nolint:forcetypeassert

		req.MetricsMode, err = supportedMetricsMode(req.MetricsMode, req.PmmAgentId)
		if err != nil {
			return err
		}

		mysqlOptions, err := models.MySQLOptionsFromRequest(req)
		if err != nil {
			return err
		}
		mysqlOptions.TableCountTablestatsGroupLimit = tablestatsGroupTableLimit

		row, err := models.CreateAgent(tx.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
			PMMAgentID:    req.PmmAgentId,
			ServiceID:     service.ServiceID,
			Username:      req.Username,
			Password:      req.Password,
			AgentPassword: req.AgentPassword,
			TLS:           req.Tls,
			TLSSkipVerify: req.TlsSkipVerify,
			MySQLOptions:  mysqlOptions,
			ExporterOptions: models.ExporterOptions{
				ExposeExporter:     req.ExposeExporter,
				PushMetrics:        isPushMode(req.MetricsMode),
				DisabledCollectors: req.DisableCollectors,
				ConnectionTimeout:  duration.OptionalFromProto(req.ConnectionTimeout),
			},
			LogLevel: services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
		})
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, row)
			if err != nil {
				return err
			}

			err = s.sib.GetInfoFromService(ctx, tx.Querier, service, row)
			if err != nil {
				return err
			}
			// GetInfoFromService updates the table count in row so, let's also update the response
			mysql.TableCount = *row.MySQLOptions.TableCount
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		mysql.MysqldExporter = agent.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert

		if req.QanMysqlPerfschema {
			row, err = models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, &models.CreateAgentParams{
				PMMAgentID:    req.PmmAgentId,
				ServiceID:     service.ServiceID,
				Username:      req.Username,
				Password:      req.Password,
				TLS:           req.Tls,
				TLSSkipVerify: req.TlsSkipVerify,
				QANOptions: models.QANOptions{
					MaxQueryLength:          req.MaxQueryLength,
					QueryExamplesDisabled:   req.DisableQueryExamples,
					CommentsParsingDisabled: req.DisableCommentsParsing,
				},
				MySQLOptions: mysqlOptions,
				LogLevel:     services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			mysql.QanMysqlPerfschema = agent.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
		}

		if req.QanMysqlSlowlog {
			row, err = models.CreateAgent(tx.Querier, models.QANMySQLSlowlogAgentType, &models.CreateAgentParams{
				PMMAgentID:    req.PmmAgentId,
				ServiceID:     service.ServiceID,
				Username:      req.Username,
				Password:      req.Password,
				TLS:           req.Tls,
				TLSSkipVerify: req.TlsSkipVerify,
				MySQLOptions:  mysqlOptions,
				QANOptions: models.QANOptions{
					MaxQueryLength:          req.MaxQueryLength,
					QueryExamplesDisabled:   req.DisableQueryExamples,
					CommentsParsingDisabled: req.DisableCommentsParsing,
					MaxQueryLogSize:         maxSlowlogFileSize,
				},
				LogLevel: services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			})
			if err != nil {
				return err
			}

			agent, err = services.ToAPIAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			mysql.QanMysqlSlowlog = agent.(*inventoryv1.QANMySQLSlowlogAgent) //nolint:forcetypeassert
		}

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	s.state.RequestStateUpdate(ctx, req.PmmAgentId)
	s.vc.RequestSoftwareVersionsUpdate()

	res := &managementv1.AddServiceResponse{
		Service: &managementv1.AddServiceResponse_Mysql{
			Mysql: mysql,
		},
	}

	return res, nil
}
