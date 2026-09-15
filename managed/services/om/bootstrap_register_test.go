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

// fakeStateUpdater records every pmmAgentID RequestStateUpdate was called
// with, in call order -- all this package's tests need from a real
// *agents.StateUpdater, which pulls in the whole agent registry to construct.
type fakeStateUpdater struct {
	requested []string
}

func (f *fakeStateUpdater) RequestStateUpdate(_ context.Context, pmmAgentID string) {
	f.requested = append(f.requested, pmmAgentID)
}

// registerTestNode creates a node with a running pmm-agent, the minimum
// registerBootstrapHost needs to succeed.
//
// :return: The node's id, and its pmm-agent's id -- registerBootstrapHost
// pushes a state update for the latter, which some tests need to assert on.
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

		err := svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret")
		require.NoError(t, err)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Equal(t, "node00-mongod", services[0].ServiceName)
		assert.Equal(t, "rs-test", services[0].ReplicationSet)
	})

	t.Run("registers every member of the same replica set under a distinct service name", func(t *testing.T) {
		// The bug this guards: naming every member's service after the shared
		// replicaSetName ("rs-test-mongod" for all three) collided on PMM's
		// instance-wide service-name uniqueness constraint, so only the first
		// host of a real 3-member run ever got registered -- the other two
		// failed "already exists" on every stepper tick, forever.
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		node00, _ := registerTestNode(t, db, "node00")
		node01, _ := registerTestNode(t, db, "node01")
		node02, _ := registerTestNode(t, db, "node02")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), node00, "node00", "rs-test", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), node01, "node01", "rs-test", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), node02, "node02", "rs-test", "admin", "secret"))

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

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret"))

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: nodeID})
		require.NoError(t, err)
		assert.Len(t, services, 1, "a second call must not create a duplicate service")
	})

	t.Run("pushes a state update for the host's pmm-agent on a fresh registration", func(t *testing.T) {
		// The bug this guards: pmm-agent only ever starts an exporter in response
		// to this push, never merely because its Agent row now exists in
		// Postgres -- confirmed against a real bootstrapped host, whose exporter
		// sat at AGENT_STATUS_UNKNOWN (and the service showed Down) forever
		// without it, even though mongod itself was up, secured, and reachable.
		db := storeTestDB(t)
		updater := &fakeStateUpdater{}
		svc := &Service{db: db, l: logrus.WithField("test", t.Name()), stateUpdater: updater}
		nodeID, pmmAgentID := registerTestNode(t, db, "node00")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret"))

		assert.Equal(t, []string{pmmAgentID}, updater.requested)
	})

	t.Run("also pushes a state update when the host was already registered", func(t *testing.T) {
		// Self-healing for a host registered before this push existed at all
		// (or by a leader that failed over mid-registration): every tick that
		// revisits a succeeded run (RunBootstrapStepper's own doc comment) gets
		// another chance to nudge pmm-agent into actually running the exporter.
		db := storeTestDB(t)
		updater := &fakeStateUpdater{}
		svc := &Service{db: db, l: logrus.WithField("test", t.Name()), stateUpdater: updater}
		nodeID, pmmAgentID := registerTestNode(t, db, "node00")

		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret"))
		require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret"))

		assert.Equal(t, []string{pmmAgentID, pmmAgentID}, updater.requested)
	})

	t.Run("does not panic when no state updater is wired", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		nodeID, _ := registerTestNode(t, db, "node00")

		assert.NotPanics(t, func() {
			require.NoError(t, svc.registerBootstrapHost(t.Context(), nodeID, "node00", "rs-test", "admin", "secret"))
		})
	})

	t.Run("fails when the node has no running pmm-agent", func(t *testing.T) {
		db := storeTestDB(t)
		svc := &Service{db: db, l: logrus.WithField("test", t.Name())}
		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "node-without-agent",
		})
		require.NoError(t, err)

		err = svc.registerBootstrapHost(t.Context(), node.NodeID, "node-without-agent", "rs-test", "admin", "secret")
		assert.Error(t, err)
	})
}
