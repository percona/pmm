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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/auth"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
)

// scopeFixture is two nodes, each with its own pmm-agent, service and exporter, so that a
// caller confined to one must not see or touch the other's.
type scopeFixture struct {
	db                 *reform.DB
	nodeA, nodeB       *models.Node
	pmmAgentA          *models.Agent
	serviceA, serviceB *models.Service
	exporterA          *models.Agent
}

func setupScopeFixture(t *testing.T) *scopeFixture {
	t.Helper()

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	f := &scopeFixture{db: db}

	var err error
	f.nodeA, err = models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "scope-node-a", Address: "10.0.0.1",
	})
	require.NoError(t, err)
	f.nodeB, err = models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "scope-node-b", Address: "10.0.0.2",
	})
	require.NoError(t, err)

	f.pmmAgentA, err = models.CreatePMMAgent(db.Querier, f.nodeA.NodeID, nil)
	require.NoError(t, err)

	f.serviceA, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "scope-svc-a", NodeID: f.nodeA.NodeID, Address: new("10.0.0.1"), Port: new(uint16(3306)),
	})
	require.NoError(t, err)
	f.serviceB, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "scope-svc-b", NodeID: f.nodeB.NodeID, Address: new("10.0.0.2"), Port: new(uint16(3306)),
	})
	require.NoError(t, err)

	f.exporterA, err = models.CreateAgent(db.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
		PMMAgentID: f.pmmAgentA.AgentID, ServiceID: f.serviceA.ServiceID,
	})
	require.NoError(t, err)

	return f
}

func TestFindNodeIDForAgent(t *testing.T) {
	f := setupScopeFixture(t)

	// A pmm-agent resolves through runs_on_node_id, an exporter through its service.
	for name, agentID := range map[string]string{
		"pmm-agent": f.pmmAgentA.AgentID,
		"exporter":  f.exporterA.AgentID,
	} {
		t.Run(name, func(t *testing.T) {
			nodeID, err := models.FindNodeIDForAgent(f.db.Querier, agentID)
			require.NoError(t, err)
			assert.Equal(t, f.nodeA.NodeID, nodeID)
		})
	}
}

func TestScopeFiltersAndChecks(t *testing.T) {
	f := setupScopeFixture(t)

	unscoped := logger.Set(t.Context(), t.Name())
	scopedToA := auth.WithNodeScope(unscoped, f.nodeA.NodeID)

	t.Run("nodes are filtered to the bound node", func(t *testing.T) {
		all, err := models.FindNodes(f.db.Querier, models.NodeFilters{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(all), 2)

		assert.Len(t, scopeNodes(unscoped, all), len(all))

		all, err = models.FindNodes(f.db.Querier, models.NodeFilters{})
		require.NoError(t, err)
		kept := scopeNodes(scopedToA, all)
		require.Len(t, kept, 1)
		assert.Equal(t, f.nodeA.NodeID, kept[0].NodeID)
	})

	t.Run("services are filtered to the bound node", func(t *testing.T) {
		all, err := models.FindServices(f.db.Querier, models.ServiceFilters{})
		require.NoError(t, err)

		kept := scopeServices(scopedToA, all)
		for _, s := range kept {
			assert.Equal(t, f.nodeA.NodeID, s.NodeID)
		}
		assert.NotEmpty(t, kept)
	})

	t.Run("agents are filtered to the bound node", func(t *testing.T) {
		all, err := models.FindAgents(f.db.Querier, models.AgentFilters{})
		require.NoError(t, err)

		kept, err := scopeAgents(scopedToA, f.db.Querier, all)
		require.NoError(t, err)

		ids := make(map[string]struct{}, len(kept))
		for _, a := range kept {
			ids[a.AgentID] = struct{}{}
		}
		assert.Contains(t, ids, f.pmmAgentA.AgentID)
		assert.Contains(t, ids, f.exporterA.AgentID)
	})

	t.Run("a node token may act on its own node and not another", func(t *testing.T) {
		require.NoError(t, auth.CheckNodeScope(scopedToA, f.nodeA.NodeID))
		assert.Equal(t, codes.PermissionDenied, status.Code(auth.CheckNodeScope(scopedToA, f.nodeB.NodeID)))

		// A human admin is unconfined.
		require.NoError(t, auth.CheckNodeScope(unscoped, f.nodeB.NodeID))
	})

	t.Run("service and agent checks follow the object's node", func(t *testing.T) {
		require.NoError(t, auth.CheckServiceScope(scopedToA, f.db.Querier, f.serviceA.ServiceID))
		assert.Equal(t, codes.PermissionDenied,
			status.Code(auth.CheckServiceScope(scopedToA, f.db.Querier, f.serviceB.ServiceID)))

		require.NoError(t, auth.CheckAgentScope(scopedToA, f.db.Querier, f.exporterA.AgentID))
	})
}

func TestCheckAddAgentScope(t *testing.T) {
	f := setupScopeFixture(t)

	as := &AgentsService{db: f.db}
	unscoped := logger.Set(t.Context(), t.Name())
	scopedToA := auth.WithNodeScope(unscoped, f.nodeA.NodeID)

	// Unconfined callers are never restricted, even with nothing to place the agent by.
	require.NoError(t, as.CheckAddAgentScope(unscoped, "", "", ""))

	require.NoError(t, as.CheckAddAgentScope(scopedToA, f.pmmAgentA.AgentID, "", ""))
	require.NoError(t, as.CheckAddAgentScope(scopedToA, "", f.serviceA.ServiceID, ""))
	require.NoError(t, as.CheckAddAgentScope(scopedToA, "", "", f.nodeA.NodeID))

	assert.Equal(t, codes.PermissionDenied, status.Code(as.CheckAddAgentScope(scopedToA, "", f.serviceB.ServiceID, "")))
	assert.Equal(t, codes.PermissionDenied, status.Code(as.CheckAddAgentScope(scopedToA, "", "", f.nodeB.NodeID)))
	// Nothing to place it by means it cannot be confined, so it is refused.
	assert.Equal(t, codes.PermissionDenied, status.Code(as.CheckAddAgentScope(scopedToA, "", "", "")))
}
