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

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"gopkg.in/reform.v1"
)

// ExternalService External Management Service.
//nolint:unused
type ExternalService struct {
	db   *reform.DB
	vmdb prometheusService
}

// NewExternalService creates new External Management Service.
func NewExternalService(db *reform.DB, vmdb prometheusService) *ExternalService {
	return &ExternalService{db: db, vmdb: vmdb}
}

func (e ExternalService) AddExternal(ctx context.Context, req *managementpb.AddExternalRequest) (*managementpb.AddExternalResponse, error) {
	res := new(managementpb.AddExternalResponse)

	if e := e.db.InTransaction(func(tx *reform.TX) error {
		service, err := models.AddNewService(tx.Querier, models.ExternalServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         req.NodeId,
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
		res.Service = invService.(*inventorypb.ExternalService)

		params := &models.CreateExternalExporterParams{
			RunsOnNodeID: req.RunsOnNodeId,
			ServiceID:    service.ServiceID,
			Username:     req.Username,
			Password:     req.Password,
			Scheme:       req.Scheme,
			MetricsPath:  req.MetricsPath,
			ListenPort:   req.ListenPort,
			CustomLabels: req.CustomLabels,
		}
		row, err := models.CreateExternalExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res.ExternalExporter = agent.(*inventorypb.ExternalExporter)

		return nil
	}); e != nil {
		return nil, e
	}

	// It's required to regenerate victoriametrics config file.
	e.vmdb.RequestConfigurationUpdate()
	return res, nil
}
