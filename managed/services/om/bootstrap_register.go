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

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// pushMetricsFor mirrors AddService's default metrics mode: push, except for pmm-agent on PMM Server.
// Pull would need PMM Server to reach the host's ephemeral exporter port.
func pushMetricsFor(pmmAgentID string) bool {
	return pmmAgentID != models.PMMServerAgentID
}

// registerBootstrapHost is idempotent. Service names are keyed on host, not replica set name,
// because names must be unique across PMM and every member shares the replica set name.
func (s *Service) registerBootstrapHost(
	ctx context.Context, nodeID, host, replicaSetName string, port uint32, environment, cluster, username, password string,
) error {
	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, nodeID)
	if err != nil {
		return fmt.Errorf("failed to find a pmm-agent on node %s: %w", nodeID, err)
	}
	if len(agents) == 0 {
		return fmt.Errorf("no pmm-agent runs on node %s; cannot register its mongod", nodeID)
	}
	pmmAgentID := agents[0].AgentID

	existing, err := models.FindServices(s.db.Querier, models.ServiceFilters{NodeID: nodeID, ServiceType: new(models.MongoDBServiceType)})
	if err != nil {
		return fmt.Errorf("failed to check for an existing MongoDB service on node %s: %w", nodeID, err)
	}
	if len(existing) > 0 {
		// Request a state update anyway: pmm-agent starts the exporter only on one, and an earlier
		// registration may have missed it.
		if s.stateUpdater != nil {
			s.stateUpdater.RequestStateUpdate(ctx, pmmAgentID)
		}
		return nil
	}

	address := nodeID
	node, findErr := models.FindNodeByID(s.db.Querier, nodeID)
	if findErr == nil && node.Address != "" {
		address = node.Address
	}

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		service, err := models.AddNewService(tx.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName:    host + "-mongod",
			NodeID:         nodeID,
			Environment:    environment,
			Cluster:        cluster,
			ReplicationSet: replicaSetName,
			Address:        &address,
			Port:           new(uint16(port)), //nolint:gosec // the API caps the port at 65535
		})
		if err != nil {
			return fmt.Errorf("failed to add the MongoDB service: %w", err)
		}

		_, err = models.CreateAgent(tx.Querier, models.MongoDBExporterType, &models.CreateAgentParams{
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
		return nil
	})
	if e != nil {
		return e
	}
	if s.stateUpdater != nil {
		s.stateUpdater.RequestStateUpdate(ctx, pmmAgentID)
	}
	return nil
}
