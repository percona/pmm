// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package inventory

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

func TestOtelCollectorDuplicateAddAndChange(t *testing.T) {
	_, as, _, teardown, ctx, _ := setup(t)
	t.Cleanup(func() { teardown(t) })

	// IsConnected is only called from toInventoryAgent for rows of type
	// PMMAgentType. This test never calls List or otherwise enumerates the
	// seeded pmm-server pmm-agent row, so the only IsConnected call that
	// actually happens is for the newly created pmm-agent (id ...005).
	// Mark the PMMServerAgentID expectation Maybe() so the mock doesn't fail
	// AssertExpectations if it's never hit, while still permitting the call
	// in case the call graph changes.
	as.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Maybe()
	as.r.(*mockAgentsRegistry).On("IsConnected", "00000000-0000-4000-8000-000000000005").Return(true)
	as.state.(*mockAgentsStateUpdater).On("RequestStateUpdate", ctx, mock.AnythingOfType("string")).Return()

	pmmAgent, err := as.AddPMMAgent(ctx, &inventoryv1.AddPMMAgentParams{
		RunsOnNodeId: models.PMMServerNodeID,
	})
	require.NoError(t, err)
	pmmAgentID := pmmAgent.GetPmmAgent().AgentId

	otelResp, err := as.AddOtelCollector(ctx, &inventoryv1.AddOtelCollectorParams{
		PmmAgentId:   pmmAgentID,
		CustomLabels: map[string]string{"tier": "test"},
	})
	require.NoError(t, err)
	otelID := otelResp.GetOtelCollector().AgentId
	require.Contains(t, otelResp.GetOtelCollector().CustomLabels, "tier")
	assert.Equal(t, "test", otelResp.GetOtelCollector().CustomLabels["tier"])

	_, err = as.AddOtelCollector(ctx, &inventoryv1.AddOtelCollectorParams{
		PmmAgentId: pmmAgentID,
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Convert(err).Code())

	ch, err := as.ChangeOtelCollector(ctx, otelID, &inventoryv1.ChangeOtelCollectorParams{
		MergeLabels: map[string]string{"extra": "1"},
		AddLogSources: []*inventoryv1.LogSource{
			{Path: "/var/log/one.log", Preset: "raw"},
			{Path: "/var/log/two.log", Preset: "raw"},
		},
	})
	require.NoError(t, err)
	labels := ch.GetOtelCollector().CustomLabels
	assert.Equal(t, "test", labels["tier"])
	assert.Equal(t, "1", labels["extra"])
	var sources []struct {
		Path   string `json:"path"`
		Preset string `json:"preset"`
	}
	require.NoError(t, json.Unmarshal([]byte(labels["log_sources"]), &sources))
	require.Len(t, sources, 2)

	ch2, err := as.ChangeOtelCollector(ctx, otelID, &inventoryv1.ChangeOtelCollectorParams{
		AddLogSources: []*inventoryv1.LogSource{
			{Path: "/var/log/one.log", Preset: "raw"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(ch2.GetOtelCollector().CustomLabels["log_sources"]), &sources))
	require.Len(t, sources, 2)

	_, err = as.ChangeOtelCollector(ctx, otelID, &inventoryv1.ChangeOtelCollectorParams{
		MergeLabels: map[string]string{"log_sources": "nope"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Convert(err).Code())

	_, err = as.ChangeOtelCollector(ctx, pmmAgentID, &inventoryv1.ChangeOtelCollectorParams{
		MergeLabels: map[string]string{"x": "y"},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Convert(err).Code())

	chEmpty, err := as.ChangeOtelCollector(ctx, otelID, &inventoryv1.ChangeOtelCollectorParams{
		MergeLabels: map[string]string{"tier": ""},
	})
	require.NoError(t, err)
	_, hasTier := chEmpty.GetOtelCollector().CustomLabels["tier"]
	assert.False(t, hasTier)

	chReplace, err := as.ChangeOtelCollector(ctx, otelID, &inventoryv1.ChangeOtelCollectorParams{
		ReplaceLogSources: true,
		SetLogSources: []*inventoryv1.LogSource{
			{Path: "/var/log/only.log", Preset: "raw"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(chReplace.GetOtelCollector().CustomLabels["log_sources"]), &sources))
	require.Len(t, sources, 1)
	assert.Equal(t, "/var/log/only.log", sources[0].Path)
}
