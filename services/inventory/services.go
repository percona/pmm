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
	db *reform.DB
	r  agentsRegistry
}

// NewServicesService creates new ServicesService
func NewServicesService(db *reform.DB, r agentsRegistry) *ServicesService {
	return &ServicesService{
		db: db,
		r:  r,
	}
}

// ServiceFilters represents filters for services list.
type ServiceFilters struct {
	// Return only Services runs on that Node.
	NodeID string
}

// List selects all Services in a stable order.
//nolint:unparam
func (ss *ServicesService) List(ctx context.Context, filters ServiceFilters) ([]inventorypb.Service, error) {
	var servicesM []*models.Service
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		switch {
		case filters.NodeID != "":
			servicesM, err = models.ServicesForNode(tx.Querier, filters.NodeID)
		default:
			servicesM, err = models.FindAllServices(tx.Querier)
		}
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
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// Both address and socket can't be empty, etc.

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
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

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
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// Both address and socket can't be empty, etc.

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

// Remove removes Service without any Agents.
//nolint:unparam
func (ss *ServicesService) Remove(ctx context.Context, id string, force bool) error {
	return ss.db.InTransaction(func(tx *reform.TX) error {
		mode := models.RemoveRestrict
		if force {
			mode = models.RemoveCascade
		}
		return models.RemoveService(tx.Querier, id, mode)
	})
}
