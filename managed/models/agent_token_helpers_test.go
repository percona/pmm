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

package models_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestAgentTokens(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	nodeA, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "token-node-a", Address: "10.1.0.1",
	})
	require.NoError(t, err)
	nodeB, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "token-node-b", Address: "10.1.0.2",
	})
	require.NoError(t, err)

	t.Run("a token resolves to its node and is stored only as a hash", func(t *testing.T) {
		row, token, err := models.CreateAgentToken(db.Querier, nodeA.NodeID)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(token, models.AgentTokenPrefix), "token must be identifiable without a round-trip")
		assert.NotContains(t, row.TokenHash, token)
		assert.Equal(t, models.HashAgentToken(token), row.TokenHash)

		nodeID, err := models.FindNodeIDByAgentToken(db.Querier, token)
		require.NoError(t, err)
		assert.Equal(t, nodeA.NodeID, nodeID)
	})

	t.Run("each node gets a distinct token", func(t *testing.T) {
		_, tokenA, err := models.CreateAgentToken(db.Querier, nodeA.NodeID)
		require.NoError(t, err)
		_, tokenB, err := models.CreateAgentToken(db.Querier, nodeB.NodeID)
		require.NoError(t, err)
		require.NotEqual(t, tokenA, tokenB)

		gotA, err := models.FindNodeIDByAgentToken(db.Querier, tokenA)
		require.NoError(t, err)
		gotB, err := models.FindNodeIDByAgentToken(db.Querier, tokenB)
		require.NoError(t, err)
		assert.Equal(t, nodeA.NodeID, gotA)
		assert.Equal(t, nodeB.NodeID, gotB)
	})

	t.Run("unknown and foreign credentials are refused", func(t *testing.T) {
		for _, token := range []string{
			models.AgentTokenPrefix + "not-a-real-token",
			"glsa_something_grafana",
			"",
		} {
			_, err := models.FindNodeIDByAgentToken(db.Querier, token)
			require.ErrorIs(t, err, models.ErrInvalidAgentToken, "token = %q", token)
		}
	})

	t.Run("revoking a node's tokens invalidates them", func(t *testing.T) {
		_, token, err := models.CreateAgentToken(db.Querier, nodeB.NodeID)
		require.NoError(t, err)

		require.NoError(t, models.RemoveAgentTokensForNode(db.Querier, nodeB.NodeID))

		_, err = models.FindNodeIDByAgentToken(db.Querier, token)
		require.ErrorIs(t, err, models.ErrInvalidAgentToken)

		// Another node's tokens survive.
		_, otherToken, err := models.CreateAgentToken(db.Querier, nodeA.NodeID)
		require.NoError(t, err)
		_, err = models.FindNodeIDByAgentToken(db.Querier, otherToken)
		require.NoError(t, err)
	})

	t.Run("removing the node revokes its tokens", func(t *testing.T) {
		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "token-node-gone", Address: "10.1.0.3",
		})
		require.NoError(t, err)
		_, token, err := models.CreateAgentToken(db.Querier, node.NodeID)
		require.NoError(t, err)

		require.NoError(t, models.RemoveNode(db.Querier, node.NodeID, models.RemoveCascade))

		_, err = models.FindNodeIDByAgentToken(db.Querier, token)
		require.ErrorIs(t, err, models.ErrInvalidAgentToken)
	})
}
