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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
)

func raiseAttention(t *testing.T, q *reform.Querier, clusterID, reason, path string) {
	t.Helper()
	var ok bool
	require.NoError(t, q.QueryRow(
		"SELECT om_attention_upsert($1::uuid, $2, 'cluster', $1::uuid, $3, '{}'::jsonb, '{}'::jsonb, now())",
		clusterID, reason, path).Scan(&ok))
	require.True(t, ok)
}

func clearAttention(t *testing.T, q *reform.Querier, clusterID, reason, path string) {
	t.Helper()
	var ok bool
	require.NoError(t, q.QueryRow(
		"SELECT om_attention_clear($1::uuid, $2, 'cluster', $1::uuid, $3, now() + interval '1 second')",
		clusterID, reason, path).Scan(&ok))
	require.True(t, ok)
}

func TestAttentionGateDefaultBlocksAny(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	raiseAttention(t, db.Querier, clusterID, AttentionDrift, "")

	block, blocked, err := checkAttentionGate(db.Querier, AttentionPolicy{}, clusterID, nil)
	require.NoError(t, err)
	require.True(t, blocked)
	assert.Equal(t, AttentionDrift, block.Reason)

	clearAttention(t, db.Querier, clusterID, AttentionDrift, "")
	_, blocked, err = checkAttentionGate(db.Querier, AttentionPolicy{}, clusterID, nil)
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestAttentionGateExemptAll(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	raiseAttention(t, db.Querier, clusterID, AttentionIdentityMismatch, "")

	_, blocked, err := checkAttentionGate(db.Querier, AttentionPolicy{ExemptAll: true}, clusterID, nil)
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestAttentionGatePerReasonExempt(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	policy := AttentionPolicy{ExemptReasons: map[string]bool{AttentionDrift: true, AttentionIdentityMismatch: true}}

	raiseAttention(t, db.Querier, clusterID, AttentionDrift, "")
	_, blocked, err := checkAttentionGate(db.Querier, policy, clusterID, nil)
	require.NoError(t, err)
	assert.False(t, blocked, "an exempt reason does not block")

	raiseAttention(t, db.Querier, clusterID, AttentionOperationFailed, "")
	block, blocked, err := checkAttentionGate(db.Querier, policy, clusterID, nil)
	require.NoError(t, err)
	require.True(t, blocked, "a non-exempt reason still blocks")
	assert.Equal(t, AttentionOperationFailed, block.Reason)
}

func TestAttentionGateDeclaredPaths(t *testing.T) {
	db := lockTestDB(t)
	clusterID := lockCluster(t, db.Querier)
	policy := AttentionPolicy{DeclaredPaths: []string{"net.port"}}

	raiseAttention(t, db.Querier, clusterID, AttentionDrift, "storage.dbPath")
	_, blocked, err := checkAttentionGate(db.Querier, policy, clusterID, nil)
	require.NoError(t, err)
	assert.False(t, blocked, "drift off a declared path does not block")

	raiseAttention(t, db.Querier, clusterID, AttentionDrift, "net.port")
	block, blocked, err := checkAttentionGate(db.Querier, policy, clusterID, nil)
	require.NoError(t, err)
	require.True(t, blocked)
	assert.Equal(t, "net.port", block.Path)
}
