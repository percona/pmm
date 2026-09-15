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

// mongodExporterPort is the packaged mongod's own unconfigured default -- see
// PackagesInstallStrategy's own module docstring on SEP's side. Bootstrap never
// makes this configurable yet, so registration never needs to ask.
const mongodExporterPort = 27017

// registerBootstrapHost registers one bootstrapped host's mongod with PMM's own
// inventory, as a MongoDB service monitored by that node's pmm-agent, authenticating
// with the one user the stepper generated for this run (RunBootstrapStepper's own
// doc comment; both the customer's root user and PMM's monitoring credential --
// PMM-15347/questions.md Q7).
//
// Deliberately not routed through ManagementService.AddService: that entry point is
// shaped for a live, user-initiated gRPC request (an operator's own auth/RBAC
// context, request-level options this internal call has none of) and its
// checkNodeIsEligible does more than this call needs. This calls the same two model
// functions addMongoDB itself calls -- models.AddNewService, models.CreateAgent --
// directly, inside one transaction, using the db handle Service already holds.
//
// Idempotent by construction, not by a "was this run registered" flag: called again
// for a host that already has a MongoDB service (the ordinary case on every tick
// after the first, since SEP never forgets a succeeded run -- see
// RunBootstrapStepper's own doc comment on why succeeded runs are revisited) it
// finds that service already exists and returns nil without creating a second one.
func (s *Service) registerBootstrapHost(ctx context.Context, nodeID, replicaSetName, username, password string) error {
	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, nodeID)
	if err != nil {
		return fmt.Errorf("failed to find a pmm-agent on node %s: %w", nodeID, err)
	}
	if len(agents) == 0 {
		return fmt.Errorf("no pmm-agent runs on node %s; cannot register its mongod", nodeID)
	}
	pmmAgentID := agents[0].AgentID

	existing, err := models.FindServices(s.db.Querier, models.ServiceFilters{NodeID: nodeID, ServiceType: pointerToServiceType(models.MongoDBServiceType)})
	if err != nil {
		return fmt.Errorf("failed to check for an existing MongoDB service on node %s: %w", nodeID, err)
	}
	if len(existing) > 0 {
		// Already registered -- a previous tick (on this leader or another, before
		// or after a failover) already did this. See the doc comment above.
		return nil
	}

	address := nodeID
	if node, findErr := models.FindNodeByID(s.db.Querier, nodeID); findErr == nil && node.Address != "" {
		address = node.Address
	}

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		service, err := models.AddNewService(tx.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName:    fmt.Sprintf("%s-mongod", replicaSetName),
			NodeID:         nodeID,
			ReplicationSet: replicaSetName,
			Address:        &address,
			Port:           pointerToPort(mongodExporterPort),
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
		})
		if err != nil {
			return fmt.Errorf("failed to create the mongodb_exporter agent: %w", err)
		}
		return nil
	})
	if e != nil {
		return e
	}
	return nil
}

// pointerToServiceType is a typed &v -- models.ServiceFilters wants a pointer to
// discriminate "any type" from "this type", and Go has no address-of-literal syntax.
func pointerToServiceType(t models.ServiceType) *models.ServiceType { return &t }

// pointerToPort is a typed &v for the same reason -- see pointerToServiceType.
func pointerToPort(port uint16) *uint16 { return &port }
