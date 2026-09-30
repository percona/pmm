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
// PackagesInstallStrategy's own module docstring on PMM Extensions' side. Bootstrap never
// makes this configurable yet, so registration never needs to ask.
const mongodExporterPort = 27017

// pushMetricsFor reports whether a bootstrapped host's exporter should push its
// metrics rather than wait to be scraped.
//
// This mirrors one decision that ManagementService.AddService makes and this
// package deliberately does not call (see registerBootstrapHost's own comment on
// why it uses the model functions directly). There, the request's metrics mode
// runs through supportedMetricsMode and isPushMode
// (services/management/management.go) before reaching models.CreateAgent; an
// unset mode resolves to push for every agent except pmm-agent on PMM Server,
// which cannot push and is sent back to pull. That is the whole rule, so it is
// reproduced here rather than exporting two helpers out of that package for one
// caller -- but the source of truth is there, and this should follow it if it
// moves.
//
// Leaving it unset, as this did, is not a neutral default. CreateAgentParams'
// zero value is pull, which makes every bootstrapped service depend on PMM
// Server being able to open a connection *to* the host's exporter port. That
// port is ephemeral -- pmm-agent allocates from ports-min 42000 upward -- so a
// deployment has to allow a range inbound to each managed host for OM's own
// bootstrap output to be monitored at all.
//
// Caught on a CHAOS deployment, where the network allows only the ports its
// terraform names (4647 for Nomad, 27017 for MongoDB). Six hosts bootstrapped
// successfully, registered correctly, reported every step green including
// confirm_monitoring -- and then sat Down forever, with no mongodb_up series at
// all, because nothing could reach :42003. The replica set deployed by
// pmm-admin on the same network was Up throughout, and the only difference
// between them in PMM's own database was push_metrics true against false.
func pushMetricsFor(pmmAgentID string) bool {
	return pmmAgentID != models.PMMServerAgentID
}

// registerBootstrapHost registers one bootstrapped host's mongod with PMM's own
// inventory, as a MongoDB service monitored by that node's pmm-agent, authenticating
// with the one user the stepper generated for this run (RunBootstrapStepper's own
// doc comment; both the customer's root user and PMM's monitoring credential --
// PMM-15347/questions.md Q7). Its environment and cluster arguments label the
// service the same way ManagementService.AddService's own fields do -- see
// completeSucceededRun's own doc comment on where they come from; either may be
// empty, same as adding a service without them today.
//
// Deliberately not routed through ManagementService.AddService: that entry point is
// shaped for a live, user-initiated gRPC request (an operator's own auth/RBAC
// context, request-level options this internal call has none of) and its
// checkNodeIsEligible does more than this call needs. This calls the same two model
// functions addMongoDB itself calls -- models.AddNewService, models.CreateAgent --
// directly, inside one transaction, using the db handle Service already holds.
//
// ServiceName is keyed on host, the executor name (e.g. "pmm-client-node00"), not
// replicaSetName: models.AddNewService enforces service names unique across the
// whole PMM instance, and every member of one replica set shares the same
// replicaSetName by definition. Confirmed against a real 3-member run: naming
// every member's service "<replicaSetName>-mongod" registered only the first
// host, leaving the other two failing "already exists" forever (every tick,
// since a failed registration is exactly the case this function retries).
// Host is what actually distinguishes them; ReplicationSet below is what
// still ties all three together as one logical set in PMM's own model.
//
// Idempotent by construction, not by a "was this run registered" flag: called again
// for a host that already has a MongoDB service (the ordinary case on every tick
// after the first, since PMM Extensions never forgets a succeeded run -- see
// RunBootstrapStepper's own doc comment on why succeeded runs are revisited) it
// finds that service already exists and returns nil without creating a second one.
//
// Also pushes pmmAgentID a state update once the transaction commits, the same
// way addMongoDB (services/management/mongodb.go) does right after its own
// models.CreateAgent call -- pmm-agent only ever starts an exporter in
// response to that push, never merely because its row now exists in Postgres.
// Confirmed against a real bootstrapped host: without this, the new
// mongodb_exporter agent sat at AGENT_STATUS_UNKNOWN and the service showed
// Down in PMM's own UI forever, even though mongod itself was up, secured,
// and reachable the whole time.
func (s *Service) registerBootstrapHost(ctx context.Context, nodeID, host, replicaSetName, environment, cluster, username, password string) error {
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
		// Already registered -- a previous tick (on this leader or another, before
		// or after a failover) already did this. See the doc comment above.
		//
		// Still pushes a state update rather than returning immediately: cheap
		// (RequestStateUpdate is a non-blocking, already-debounced channel send)
		// and self-healing for a host whose earlier registration never got one --
		// exactly what a host registered before this function pushed updates at
		// all is stuck in otherwise, permanently, with no other path back to a
		// running exporter.
		//
		// Deliberately does *not* do the same for ExporterOptions.PushMetrics,
		// though this is the only place that could. A host registered before
		// pushMetricsFor existed keeps push_metrics false for good: its run is
		// marked by MarkOmBootstrapRunRegistered, stepBootstrapRuns then drops it
		// from the sweep, and nothing else revisits the agent row. That gap is
		// real but its population is empty -- no PMM release before this one ever
		// ran registerBootstrapHost, so the only hosts that can carry the old
		// value are ones a pre-merge build of this branch registered on a dev or
		// demo deployment. Repair those with `pmm-admin inventory change agent
		// mongodb-exporter --push-metrics <agent-id>` rather than carrying
		// reconcile code whose target set is empty from the day this merges.
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
			Port:           new(uint16(mongodExporterPort)),
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
