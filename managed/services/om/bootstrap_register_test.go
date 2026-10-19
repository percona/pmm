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
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type fakeStateUpdater struct {
	requested []string
}

func (f *fakeStateUpdater) RequestStateUpdate(_ context.Context, pmmAgentID string) {
	f.requested = append(f.requested, pmmAgentID)
}

// registerTestNode creates a node with a running pmm-agent and returns the node and pmm-agent IDs.
func registerTestNode(t *testing.T, db *reform.DB, nodeName string) (string, string) {
	t.Helper()
	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: nodeName,
	})
	require.NoError(t, err)
	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)
	return node.NodeID, pmmAgent.AgentID
}

func TestRegisterBootstrapHost(t *testing.T) {
	t.Run("registers a service and an exporter agent for the node", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		err := svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret")
		require.NoError(t, err)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Equal(t, "node00-mongod", services[0].ServiceName)
		assert.Equal(t, "rs-test", services[0].ReplicationSet)
	})

	t.Run("registers the service on the port mongod was installed on", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		err := svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27018, "", "", "admin", "secret")
		require.NoError(t, err)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		require.Len(t, services, 1)
		require.NotNil(t, services[0].Port)
		assert.Equal(t, uint16(27018), *services[0].Port)
	})

	t.Run("labels the service with the given environment and cluster", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		err := svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "staging", "orders", "admin", "secret")
		require.NoError(t, err)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Equal(t, "staging", services[0].Environment)
		assert.Equal(t, "orders", services[0].Cluster)
	})

	t.Run("leaves the service unlabelled when environment and cluster are blank", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		err := svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret")
		require.NoError(t, err)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Empty(t, services[0].Environment)
		assert.Empty(t, services[0].Cluster)
	})

	t.Run("registers every member of the same replica set under a distinct service name", func(t *testing.T) {
		// Naming every member after the shared replica set collides on PMM's service-name uniqueness.
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		node00, _ := registerTestNode(t, db, "node00")
		node01, _ := registerTestNode(t, db, "node01")
		node02, _ := registerTestNode(t, db, "node02")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), node00, "node00", "rs-test", 27017, "", "", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), node01, "node01", "rs-test", 27017, "", "", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), node02, "node02", "rs-test", 27017, "", "", "admin", "secret"))

		for _, nodeID := range []string{node00, node01, node02} {
			services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
			require.NoError(t, err)
			require.Lenf(t, services, 1, "node %s should have exactly one registered service", nodeID)
			assert.Equal(t, "rs-test", services[0].ReplicationSet)
		}
	})

	t.Run("is idempotent: a second call for an already-registered host is a no-op", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		assert.Len(t, services, 1, "a second call must not create a duplicate service")
	})

	t.Run("gives the exporter push metrics mode", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		require.Len(t, services, 1)
		agents, err := models.FindAgents(db.Querier, models.AgentFilters{ServiceID: services[0].ServiceID})
		require.NoError(t, err)
		require.Len(t, agents, 1)
		assert.True(t, agents[0].ExporterOptions.PushMetrics,
			"a bootstrapped host's exporter must push; pull needs inbound reachability OM cannot assume")
	})

	t.Run("keeps pull mode for an exporter on PMM Server's own agent", func(t *testing.T) {
		// supportedMetricsMode refuses push for pmm-agent on PMM Server.
		assert.False(t, pushMetricsFor(models.PMMServerAgentID))
		assert.True(t, pushMetricsFor("some-other-agent-id"))
	})

	t.Run("pushes a state update for the host's pmm-agent on a fresh registration", func(t *testing.T) {
		// pmm-agent starts an exporter only in response to this push, not because its Agent row exists.
		db := storeTestDB(t)
		updater := &fakeStateUpdater{}
		svc := &Service{db: db, l: logrus.WithField("test", t.Name()), stateUpdater: updater}
		nodeID, pmmAgentID := registerTestNode(t, db, "node00")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))

		assert.Equal(t, []string{pmmAgentID}, updater.requested)
	})

	t.Run("also pushes a state update when the host was already registered", func(t *testing.T) {
		// Self-healing: each tick revisiting a succeeded run pushes again, e.g. after a failover mid-registration.
		db := storeTestDB(t)
		updater := &fakeStateUpdater{}
		svc := &Service{db: db, l: logrus.WithField("test", t.Name()), stateUpdater: updater}
		nodeID, pmmAgentID := registerTestNode(t, db, "node00")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))

		assert.Equal(t, []string{pmmAgentID, pmmAgentID}, updater.requested)
	})

	t.Run("does not panic when no state updater is wired", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		assert.NotPanics(t, func() {
			require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", 27017, "", "", "admin", "secret"))
		})
	})

	t.Run("fails when the node has no running pmm-agent", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "node-without-agent",
		})
		require.NoError(t, err)

		err = svc.registerBootstrapHost(t.Context(), node.NodeID, "node-without-agent", "rs-test", 27017, "", "", "admin", "secret")
		assert.Error(t, err)
	})
}
