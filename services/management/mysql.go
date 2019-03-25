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
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/percona/pmm/api/managementpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/services/inventory"
)

// MySQLService MySQL Management Service.
type MySQLService struct {
	db          *reform.DB
	servicesSvc *inventory.ServicesService
	agentsSvc   *inventory.AgentsService
}

// NewMySQLService creates new MySQL Management Service.
func NewMySQLService(db *reform.DB, s *inventory.ServicesService, a *inventory.AgentsService) *MySQLService {
	return &MySQLService{db, s, a}
}

// Add adds "MySQL Service", "MySQL Exporter Agent" and "QAN MySQL PerfSchema Agent".
func (s *MySQLService) Add(ctx context.Context, req *managementpb.AddMySQLRequest) (res *managementpb.AddMySQLResponse, err error) {
	res = &managementpb.AddMySQLResponse{}

	if e := s.db.InTransaction(func(tx *reform.TX) error {
		service, err := s.servicesSvc.AddMySQL(ctx, tx.Querier, &inventory.AddDBMSServiceParams{
			ServiceName: req.ServiceName,
			NodeID:      req.NodeId,
			Address:     pointer.ToStringOrNil(req.Address),
			Port:        pointer.ToUint16OrNil(uint16(req.Port)),
		})
		if err != nil {
			return err
		}
		res.Service = service

		if req.MysqldExporter {
			request := &inventorypb.AddMySQLdExporterRequest{
				PmmAgentId: req.PmmAgentId,
				ServiceId:  service.ID(),
				Username:   req.Username,
				Password:   req.Password,
			}

			agent, err := s.agentsSvc.AddMySQLdExporter(ctx, tx.Querier, request)
			if err != nil {
				return err
			}

			res.MysqldExporter = agent
		}

		if req.QanMysqlPerfschema {
			request := &inventorypb.AddQANMySQLPerfSchemaAgentRequest{
				PmmAgentId: req.PmmAgentId,
				ServiceId:  service.ID(),
				Username:   req.QanUsername,
				Password:   req.QanPassword,
			}

			qAgent, err := s.agentsSvc.AddQANMySQLPerfSchemaAgent(ctx, tx.Querier, request)
			if err != nil {
				return err
			}

			res.QanMysqlPerfschema = qAgent
		}

		return nil
	}); e != nil {
		return nil, e
	}

	return res, nil
}
