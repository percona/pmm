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

package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
)

// An agent's ID is asserted by whoever opens the stream, so before PMM issued its own agent
// tokens there was nothing to check it against. These are the rules that close that.
func TestRegistryCheckAgentIdentity(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	ctx := logger.Set(t.Context(), t.Name())
	r := &Registry{db: db}

	nodeA, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "identity-node-a", Address: "10.5.0.1",
	})
	require.NoError(t, err)
	nodeB, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "identity-node-b", Address: "10.5.0.2",
	})
	require.NoError(t, err)

	agentA, err := models.CreatePMMAgent(db.Querier, nodeA.NodeID, nil)
	require.NoError(t, err)
	agentB, err := models.CreatePMMAgent(db.Querier, nodeB.NodeID, nil)
	require.NoError(t, err)

	_, tokenA, err := models.CreateAgentToken(db.Querier, nodeA.NodeID)
	require.NoError(t, err)

	bearer := func(token string) metadata.MD {
		return metadata.New(map[string]string{"Authorization": "Bearer " + token})
	}

	t.Run("a node token may act as its own agent", func(t *testing.T) {
		c := metadata.NewIncomingContext(ctx, bearer(tokenA))
		require.NoError(t, r.checkAgentIdentity(c, db.Querier, agentA.AgentID, nodeA.NodeID))
	})

	t.Run("a node token may not act as another node's agent", func(t *testing.T) {
		c := metadata.NewIncomingContext(ctx, bearer(tokenA))
		err := r.checkAgentIdentity(c, db.Querier, agentB.AgentID, nodeB.NodeID)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("an unknown PMM token is refused", func(t *testing.T) {
		c := metadata.NewIncomingContext(ctx, bearer(models.AgentTokenPrefix+"nope"))
		err := r.checkAgentIdentity(c, db.Querier, agentA.AgentID, nodeA.NodeID)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("no credentials cannot act as a remote node's agent", func(t *testing.T) {
		err := r.checkAgentIdentity(ctx, db.Querier, agentA.AgentID, nodeA.NodeID)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("no credentials may still act as PMM Server's own agent", func(t *testing.T) {
		serverNode, err := models.FindNodeByID(db.Querier, models.PMMServerNodeID)
		require.NoError(t, err)
		require.True(t, serverNode.IsPMMServerNode, "fixture must flag the PMM Server node")

		require.NoError(t, r.checkAgentIdentity(ctx, db.Querier, models.PMMServerAgentID, serverNode.NodeID))
	})

	t.Run("a legacy Grafana token is left to the auth layer", func(t *testing.T) {
		c := metadata.NewIncomingContext(ctx, bearer("glsa_legacy_grafana_token"))
		require.NoError(t, r.checkAgentIdentity(c, db.Querier, agentA.AgentID, nodeA.NodeID))
	})
}
