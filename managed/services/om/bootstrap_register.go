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

package om

import (
	"context"
	"fmt"
	"math"

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// pushMetricsFor mirrors AddService's default metrics mode: push, except for pmm-agent on PMM Server.
// Pull would need PMM Server to reach the host's ephemeral exporter port.
func pushMetricsFor(pmmAgentID string) bool {
	return pmmAgentID != models.PMMServerAgentID
}

// registeredMongod identifies the PMM inventory records a bootstrap host registration produced, so
// the caller can link the operation's cluster member to them.
type registeredMongod struct {
	nodeID          string
	serviceID       string
	exporterAgentID string
}

// registerBootstrapHost is idempotent. Service names are keyed on host, not replica set name,
// because names must be unique across PMM and every member shares the replica set name.
func (s *Service) registerBootstrapHost(
	ctx context.Context, nodeID, host, replicaSetName string, port uint32, environment, cluster, username, password string,
) (registeredMongod, error) {
	if port == 0 || port > math.MaxUint16 {
		return registeredMongod{}, fmt.Errorf("the run reports an invalid port %d; cannot register the mongod on node %s", port, nodeID)
	}
	servicePort := uint16(port)

	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, nodeID)
	if err != nil {
		return registeredMongod{}, fmt.Errorf("failed to find a pmm-agent on node %s: %w", nodeID, err)
	}
	if len(agents) == 0 {
		return registeredMongod{}, fmt.Errorf("no pmm-agent runs on node %s; cannot register its mongod", nodeID)
	}
	pmmAgentID := agents[0].AgentID

	existing, err := models.FindServices(s.db.Querier, models.ServiceFilters{NodeID: nodeID, ServiceType: new(models.MongoDBServiceType)})
	if err != nil {
		return registeredMongod{}, fmt.Errorf("failed to check for an existing MongoDB service on node %s: %w", nodeID, err)
	}
	if len(existing) > 0 {
		// Request a state update anyway: pmm-agent starts the exporter only on one, and an earlier
		// registration may have missed it.
		if s.stateUpdater != nil {
			s.stateUpdater.RequestStateUpdate(ctx, pmmAgentID)
		}
		return s.mongodRefs(nodeID, existing[0].ServiceID)
	}

	address := nodeID
	node, findErr := models.FindNodeByID(s.db.Querier, nodeID)
	if findErr == nil && node.Address != "" {
		address = node.Address
	}

	refs := registeredMongod{nodeID: nodeID}
	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		service, err := models.AddNewService(tx.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName:    host + "-mongod",
			NodeID:         nodeID,
			Environment:    environment,
			Cluster:        cluster,
			ReplicationSet: replicaSetName,
			Address:        &address,
			Port:           &servicePort,
		})
		if err != nil {
			return fmt.Errorf("failed to add the MongoDB service: %w", err)
		}
		refs.serviceID = service.ServiceID

		agent, err := models.CreateAgent(tx.Querier, models.MongoDBExporterType, &models.CreateAgentParams{
			PMMAgentID:          pmmAgentID,
			ServiceID:           service.ServiceID,
			Username:            username,
			Password:            password,
			MongoDBOptions:      models.MongoDBOptions{},
			SkipConnectionCheck: true,
			ExporterOptions: models.ExporterOptions{
				PushMetrics: pushMetricsFor(pmmAgentID),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create the mongodb_exporter agent: %w", err)
		}
		refs.exporterAgentID = agent.AgentID
		return nil
	})
	if e != nil {
		return registeredMongod{}, e
	}
	if s.stateUpdater != nil {
		s.stateUpdater.RequestStateUpdate(ctx, pmmAgentID)
	}
	return refs, nil
}

// mongodRefs returns the ids to link a member to for a mongod service that already exists on the node.
func (s *Service) mongodRefs(nodeID, serviceID string) (registeredMongod, error) {
	exporters, err := models.FindAgents(s.db.Querier, models.AgentFilters{ServiceID: serviceID, AgentType: new(models.MongoDBExporterType)})
	if err != nil {
		return registeredMongod{}, fmt.Errorf("failed to find the mongodb_exporter on service %s: %w", serviceID, err)
	}
	refs := registeredMongod{nodeID: nodeID, serviceID: serviceID}
	if len(exporters) > 0 {
		refs.exporterAgentID = exporters[0].AgentID
	}
	return refs, nil
}
