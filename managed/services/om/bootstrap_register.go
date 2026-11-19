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
	"slices"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// pushMetricsFor mirrors AddService's default metrics mode: push, except for pmm-agent on PMM Server.
// Pull would need PMM Server to reach the host's ephemeral exporter port.
func pushMetricsFor(pmmAgentID string) bool {
	return pmmAgentID != models.PMMServerAgentID
}

// bootstrapServiceName keys an install's service on its host rather than its replica set name,
// because service names are unique across PMM and every member shares the replica set name.
func bootstrapServiceName(host string) string {
	return host + "-mongod"
}

// hasBootstrapService reports whether the inventory app lists the service an install registers on executorHost.
func hasBootstrapService(host extensionsHost, executorHost string, port uint32) bool {
	name := bootstrapServiceName(executorHost)
	return slices.ContainsFunc(host.Services, func(service extensionsService) bool {
		return service.Name == name && service.Port != nil && int64(*service.Port) == int64(port)
	})
}

// registerBootstrapHost is idempotent.
func (s *Service) registerBootstrapHost(
	ctx context.Context, nodeID, host, replicaSetName string, port uint32, environment, cluster, username, password string,
) error {
	if port == 0 || port > math.MaxUint16 {
		return fmt.Errorf("the run reports an invalid port %d; cannot register the mongod on node %s", port, nodeID)
	}
	servicePort := uint16(port)

	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, nodeID)
	if err != nil {
		return fmt.Errorf("failed to find a pmm-agent on node %s: %w", nodeID, err)
	}
	if len(agents) == 0 {
		return fmt.Errorf("no pmm-agent runs on node %s; cannot register its mongod", nodeID)
	}
	pmmAgentID := agents[0].AgentID

	name := bootstrapServiceName(host)
	existing, err := models.FindServiceByName(s.db.Querier, name)
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to look up service %s: %w", name, err)
	}
	if err == nil {
		existingPort := pointer.GetUint16(existing.Port)
		if existing.NodeID != nodeID || existing.ServiceType != models.MongoDBServiceType || existingPort != servicePort {
			return fmt.Errorf("service %s already exists as a %s service on node %s, port %d, not this install's mongod on node %s, port %d",
				name, existing.ServiceType, existing.NodeID, existingPort, nodeID, servicePort)
		}
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
			ServiceName:    name,
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
