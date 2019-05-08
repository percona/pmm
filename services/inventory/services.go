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
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// ServicesService works with inventory API Services.
type ServicesService struct {
	db *reform.DB
	r  registry
}

// NewServicesService creates new ServicesService
func NewServicesService(db *reform.DB, r registry) *ServicesService {
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
	services := make([]*models.Service, 0)
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		switch {
		case filters.NodeID != "":
			services, err = models.ServicesForNode(tx.Querier, filters.NodeID)
		default:
			services, err = models.FindAllServices(tx.Querier)
		}
		if err != nil {
			return err
		}
		return nil
	})
	if e != nil {
		return nil, e
	}

	return ToInventoryServices(services)
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

	return ToInventoryService(service)
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

	res, err := ToInventoryService(service)
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

	res, err := ToInventoryService(service)
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

	res, err := ToInventoryService(service)
	if err != nil {
		return nil, err
	}
	return res.(*inventorypb.PostgreSQLService), nil
}

// Remove removes Service without any Agents.
//nolint:unparam
func (ss *ServicesService) Remove(ctx context.Context, id string) error {
	return ss.db.InTransaction(func(tx *reform.TX) error {
		return models.RemoveService(tx.Querier, id)
	})
}

// ToInventoryService converts database row to Inventory API Service.
func ToInventoryService(row *models.Service) (inventorypb.Service, error) {
	labels, err := row.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch row.ServiceType {
	case models.MySQLServiceType:
		return &inventorypb.MySQLService{
			ServiceId:    row.ServiceID,
			ServiceName:  row.ServiceName,
			NodeId:       row.NodeID,
			Address:      pointer.GetString(row.Address),
			Port:         uint32(pointer.GetUint16(row.Port)),
			CustomLabels: labels,
		}, nil
	case models.MongoDBServiceType:
		return &inventorypb.MongoDBService{
			ServiceId:    row.ServiceID,
			ServiceName:  row.ServiceName,
			NodeId:       row.NodeID,
			Address:      pointer.GetString(row.Address),
			Port:         uint32(pointer.GetUint16(row.Port)),
			CustomLabels: labels,
		}, nil
	case models.PostgreSQLServiceType:
		return &inventorypb.PostgreSQLService{
			ServiceId:    row.ServiceID,
			ServiceName:  row.ServiceName,
			NodeId:       row.NodeID,
			Address:      pointer.GetString(row.Address),
			Port:         uint32(pointer.GetUint16(row.Port)),
			CustomLabels: labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Service type %s", row.ServiceType))
	}
}

// ToInventoryServices converts database rows to Inventory API Services.
func ToInventoryServices(services []*models.Service) ([]inventorypb.Service, error) {
	var err error
	res := make([]inventorypb.Service, len(services))
	for i, srv := range services {
		res[i], err = ToInventoryService(srv)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
