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

package inventory

import (
	"context"

	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/services/management/common"
)

// ServicesService works with inventory API Services.
type ServicesService struct {
	db    *reform.DB
	r     agentsRegistry
	state agentsStateUpdater
	vmdb  prometheusService
	vc    versionCache
}

// NewServicesService creates new ServicesService.
func NewServicesService(
	db *reform.DB,
	r agentsRegistry,
	state agentsStateUpdater,
	vmdb prometheusService,
	vc versionCache,
) *ServicesService {
	return &ServicesService{
		db:    db,
		r:     r,
		state: state,
		vmdb:  vmdb,
		vc:    vc,
	}
}

// List selects all Services in a stable order.
//
//nolint:unparam
func (ss *ServicesService) List(_ context.Context, filters models.ServiceFilters) ([]inventorypb.Service, error) {
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

// ListActiveServiceTypes lists all active Service Types
//
//nolint:unparam
func (ss *ServicesService) ListActiveServiceTypes(_ context.Context) ([]inventorypb.ServiceType, error) {
	var types []models.ServiceType
	e := ss.db.InTransaction(func(tx *reform.TX) error {
		var err error
		types, err = models.FindActiveServiceTypes(tx.Querier)
		return err
	})
	if e != nil {
		return nil, e
	}

	res := make([]inventorypb.ServiceType, 0, len(types))
	for _, t := range types {
		switch t {
		case models.MySQLServiceType:
			res = append(res, inventorypb.ServiceType_MYSQL_SERVICE) //nolint:nosnakecase
		case models.MongoDBServiceType:
			res = append(res, inventorypb.ServiceType_MONGODB_SERVICE) //nolint:nosnakecase
		case models.PostgreSQLServiceType:
			res = append(res, inventorypb.ServiceType_POSTGRESQL_SERVICE) //nolint:nosnakecase
		case models.ProxySQLServiceType:
			res = append(res, inventorypb.ServiceType_PROXYSQL_SERVICE) //nolint:nosnakecase
		case models.HAProxyServiceType:
			res = append(res, inventorypb.ServiceType_HAPROXY_SERVICE) //nolint:nosnakecase
		case models.ExternalServiceType:
			res = append(res, inventorypb.ServiceType_EXTERNAL_SERVICE) //nolint:nosnakecase
		}
	}
	return res, nil
}

// Get selects a single Service by ID.
func (ss *ServicesService) Get(_ context.Context, id string) (inventorypb.Service, error) { //nolint:ireturn
	service := &models.Service{}
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
//
//nolint:dupl,unparam
func (ss *ServicesService) AddMySQL(_ context.Context, params *models.AddDBMSServiceParams) (*inventorypb.MySQLService, error) {
	service := &models.Service{}
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

	ss.vc.RequestSoftwareVersionsUpdate()

	return res.(*inventorypb.MySQLService), nil //nolint:forcetypeassert
}

// AddMongoDB inserts MongoDB Service with given parameters.
//
//nolint:dupl,unparam
func (ss *ServicesService) AddMongoDB(_ context.Context, params *models.AddDBMSServiceParams) (*inventorypb.MongoDBService, error) {
	service := &models.Service{}
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
	return res.(*inventorypb.MongoDBService), nil //nolint:forcetypeassert
}

// AddPostgreSQL inserts PostgreSQL Service with given parameters.
//
//nolint:dupl,unparam
func (ss *ServicesService) AddPostgreSQL(_ context.Context, params *models.AddDBMSServiceParams) (*inventorypb.PostgreSQLService, error) {
	service := &models.Service{}
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
	return res.(*inventorypb.PostgreSQLService), nil //nolint:forcetypeassert
}

// AddProxySQL inserts ProxySQL Service with given parameters.
//
//nolint:dupl,unparam
func (ss *ServicesService) AddProxySQL(_ context.Context, params *models.AddDBMSServiceParams) (*inventorypb.ProxySQLService, error) {
	service := &models.Service{}
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
	return res.(*inventorypb.ProxySQLService), nil //nolint:forcetypeassert
}

// AddHAProxyService inserts HAProxy Service with given parameters.
func (ss *ServicesService) AddHAProxyService(_ context.Context, params *models.AddDBMSServiceParams) (*inventorypb.HAProxyService, error) {
	service := &models.Service{}
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
	return res.(*inventorypb.HAProxyService), nil //nolint:forcetypeassert
}

// AddExternalService inserts External Service with given parameters.
//
//nolint:dupl,unparam
func (ss *ServicesService) AddExternalService(_ context.Context, params *models.AddDBMSServiceParams) (*inventorypb.ExternalService, error) {
	service := &models.Service{}
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
	return res.(*inventorypb.ExternalService), nil //nolint:forcetypeassert
}

// Remove removes Service without any Agents.
// Removes Service with the Agents if force == true.
// Returns an error if force == false and Service has Agents.
func (ss *ServicesService) Remove(ctx context.Context, id string, force bool) error {
	pmmAgentIds := make(map[string]struct{})

	if e := ss.db.InTransaction(func(tx *reform.TX) error {
		service, err := models.FindServiceByID(tx.Querier, id)
		if err != nil {
			return err
		}

		mode := models.RemoveRestrict
		if force {
			mode = models.RemoveCascade

			agents, err := models.FindPMMAgentsForService(tx.Querier, id)
			if err != nil {
				return err
			}

			for _, agent := range agents {
				pmmAgentIds[agent.AgentID] = struct{}{}
			}
		}

		err = models.RemoveService(tx.Querier, id, mode)
		if err != nil {
			return err
		}

		if force {
			node, err := models.FindNodeByID(tx.Querier, service.NodeID)
			if err != nil {
				return err
			}

			// For RDS and Azure remove also node.
			if node.NodeType == models.RemoteRDSNodeType || node.NodeType == models.RemoteAzureDatabaseNodeType {
				agents, err := models.FindAgents(tx.Querier, models.AgentFilters{NodeID: node.NodeID})
				if err != nil {
					return err
				}
				for _, agent := range agents {
					if agent.PMMAgentID != nil {
						pmmAgentIds[pointer.GetString(agent.PMMAgentID)] = struct{}{}
					}
				}

				if len(pmmAgentIds) <= 1 {
					if err = models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}); e != nil {
		return e
	}

	for pmmAgentID := range pmmAgentIds {
		ss.state.RequestStateUpdate(ctx, pmmAgentID)
	}

	if force {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		ss.vmdb.RequestConfigurationUpdate()
	}

	return nil
}

// AddCustomLabels adds or replaces (if key exists) custom labels for a service.
func (ss *ServicesService) AddCustomLabels(ctx context.Context, req *inventorypb.AddCustomLabelsRequest) (*inventorypb.AddCustomLabelsResponse, error) {
	errTx := ss.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		labels, err := service.GetCustomLabels()
		if err != nil {
			return err
		}
		if labels == nil {
			labels = req.CustomLabels
		} else {
			for k, v := range req.CustomLabels {
				labels[k] = v
			}
		}

		err = service.SetCustomLabels(labels)
		if err != nil {
			return err
		}

		err = tx.UpdateColumns(service, "custom_labels")
		if err != nil {
			return err
		}

		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	if err := ss.updateScrapeConfig(ctx, req.ServiceId); err != nil {
		return nil, err
	}

	return &inventorypb.AddCustomLabelsResponse{}, nil
}

// RemoveCustomLabels removes custom labels from a service.
func (ss *ServicesService) RemoveCustomLabels(ctx context.Context, req *inventorypb.RemoveCustomLabelsRequest) (*inventorypb.RemoveCustomLabelsResponse, error) {
	errTx := ss.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
		if err != nil {
			return err
		}

		labels, err := service.GetCustomLabels()
		if err != nil {
			return err
		}
		if labels == nil {
			return nil
		}

		for _, k := range req.CustomLabelKeys {
			delete(labels, k)
		}

		err = service.SetCustomLabels(labels)
		if err != nil {
			return err
		}

		err = tx.UpdateColumns(service, "custom_labels")
		if err != nil {
			return err
		}

		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	if err := ss.updateScrapeConfig(ctx, req.ServiceId); err != nil {
		return nil, err
	}

	return &inventorypb.RemoveCustomLabelsResponse{}, nil
}

// ChangeService changes service configuration.
func (ss *ServicesService) ChangeService(ctx context.Context, mgmtServices common.MgmtServices, params *models.ChangeStandardLabelsParams) error {
	if err := mgmtServices.RemoveScheduledTasks(ctx, ss.db, params); err != nil {
		return err
	}

	errTx := ss.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		err := models.ChangeStandardLabels(tx.Querier, params.ServiceID, models.ServiceStandardLabelsParams{
			Cluster:        params.Cluster,
			Environment:    params.Environment,
			ReplicationSet: params.ReplicationSet,
			ExternalGroup:  params.ExternalGroup,
		})
		return err
	})
	if errTx != nil {
		return errTx
	}

	if err := ss.updateScrapeConfig(ctx, params.ServiceID); err != nil {
		return err
	}

	return nil
}

func (ss *ServicesService) updateScrapeConfig(ctx context.Context, serviceID string) error {
	ss.vmdb.RequestConfigurationUpdate()

	agents, err := models.FindPMMAgentsForService(ss.db.Querier, serviceID)
	if err != nil {
		return err
	}

	for _, a := range agents {
		ss.state.RequestStateUpdate(ctx, a.AgentID)
	}

	return nil
}
