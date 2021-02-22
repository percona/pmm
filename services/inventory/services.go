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

package inventory

import (
	"context"

	"github.com/percona/pmm/api/inventorypb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"
)

// ServicesService works with inventory API Services.
type ServicesService struct {
	db   *reform.DB
	r    agentsRegistry
	vmdb prometheusService
}

// NewServicesService creates new ServicesService
func NewServicesService(db *reform.DB, r agentsRegistry, vmdb prometheusService) *ServicesService {
	return &ServicesService{
		db:   db,
		r:    r,
		vmdb: vmdb,
	}
}

// List selects all Services in a stable order.
//nolint:unparam
func (ss *ServicesService) List(ctx context.Context, filters models.ServiceFilters) ([]inventorypb.Service, error) {
	var servicesM []*models.Service
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		servicesM, err = models.FindServices(tx.Querier, filters)
		return err
	})
	if e != nil {
		return nil, e
	}

	res := make([]inventorypb.Service, len(servicesM))
	for i, s := range servicesM {
		res[i], e = services.ToAPIService(s)
		if e != nil {
			return nil, e
		}
	}
	return res, nil
}

// Get selects a single Service by ID.
//nolint:unparam
func (ss *ServicesService) Get(ctx context.Context, id string) (inventorypb.Service, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.FindServiceByID(tx.Querier, id)
		if err != nil {
			return err
		}
		return nil
	})

	if e != nil {
		return nil, e
	}

	return services.ToAPIService(service)
}

// AddMySQL inserts MySQL Service with given parameters.
//nolint:dupl,unparam
func (ss *ServicesService) AddMySQL(ctx context.Context, params *models.AddDBMSServiceParams) (*inventorypb.MySQLService, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.AddNewService(tx.Querier, models.MySQLServiceType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	res, err := services.ToAPIService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.MySQLService), nil
}

// AddMongoDB inserts MongoDB Service with given parameters.
//nolint:dupl,unparam
func (ss *ServicesService) AddMongoDB(ctx context.Context, params *models.AddDBMSServiceParams) (*inventorypb.MongoDBService, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.AddNewService(tx.Querier, models.MongoDBServiceType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	res, err := services.ToAPIService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.MongoDBService), nil
}

// AddPostgreSQL inserts PostgreSQL Service with given parameters.
//nolint:dupl,unparam
func (ss *ServicesService) AddPostgreSQL(ctx context.Context, params *models.AddDBMSServiceParams) (*inventorypb.PostgreSQLService, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.AddNewService(tx.Querier, models.PostgreSQLServiceType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	res, err := services.ToAPIService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.PostgreSQLService), nil
}

// AddProxySQL inserts ProxySQL Service with given parameters.
//nolint:dupl,unparam
func (ss *ServicesService) AddProxySQL(ctx context.Context, params *models.AddDBMSServiceParams) (*inventorypb.ProxySQLService, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.AddNewService(tx.Querier, models.ProxySQLServiceType, params)
		return err
	})
	if e != nil {
		return nil, e
	}

	res, err := services.ToAPIService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.ProxySQLService), nil
}

// AddHAProxyService inserts HAProxy Service with given parameters.
func (ss *ServicesService) AddHAProxyService(ctx context.Context, params *models.AddDBMSServiceParams) (*inventorypb.HAProxyService, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.AddNewService(tx.Querier, models.HAProxyServiceType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	res, err := services.ToAPIService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.HAProxyService), nil
}

// AddExternalService inserts External Service with given parameters.
//nolint:dupl,unparam
func (ss *ServicesService) AddExternalService(ctx context.Context, params *models.AddDBMSServiceParams) (*inventorypb.ExternalService, error) {
	service := new(models.Service)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		service, err = models.AddNewService(tx.Querier, models.ExternalServiceType, params)
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	res, err := services.ToAPIService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.ExternalService), nil
}

// Remove removes Service without any Agents.
// Removes Service with the Agents if force == true.
// Returns an error if force == false and Service has Agents.
func (ss *ServicesService) Remove(ctx context.Context, id string, force bool) error {
	var agents []*models.Agent

	if e := ss.db.InTransaction(func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if force {
			mode = models.RemoveCascade

			foundAgents, err := models.FindPMMAgentsForService(tx.Querier, id)
			if err != nil {
				return err
			}

			agents = foundAgents
		}

		return models.RemoveService(tx.Querier, id, mode)
	}); e != nil {
		return e
	}

	for _, a := range agents {
		ss.r.RequestStateUpdate(ctx, a.AgentID)
	}

	if force {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		ss.vmdb.RequestConfigurationUpdate()
	}

	return nil
}
