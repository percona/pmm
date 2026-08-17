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

package ha

import (
	"sync"
	"testing"

	"github.com/hashicorp/memberlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hav1beta1 "github.com/percona/pmm/api/ha/v1beta1"
	"github.com/percona/pmm/managed/models"
)

func TestHAServer_Status(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		haEnabled      bool
		expectedStatus string
	}{
		{
			name:           "HA Enabled",
			haEnabled:      true,
			expectedStatus: "Enabled",
		},
		{
			name:           "HA Disabled",
			haEnabled:      false,
			expectedStatus: "Disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &Service{
				params: &models.HAParams{
					Enabled: tt.haEnabled,
				},
			}

			server := NewHAServer(service)

			resp, err := server.Status(t.Context(), &hav1beta1.StatusRequest{})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.expectedStatus, resp.Status)
		})
	}
}

func TestHAServer_ListNodes_HADisabled(t *testing.T) {
	t.Parallel()

	service := &Service{
		params: &models.HAParams{
			Enabled: false,
		},
	}

	server := NewHAServer(service)

	resp, err := server.ListNodes(t.Context(), &hav1beta1.ListNodesRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Nodes)
	assert.Equal(t, int32(0), resp.ExpectedNodes)
}

func TestHAServer_ListNodes_NilMemberlist(t *testing.T) {
	t.Parallel()

	service := &Service{
		params: &models.HAParams{
			Enabled: true,
		},
		memberlist: nil,
		rw:         sync.RWMutex{},
	}

	server := NewHAServer(service)

	resp, err := server.ListNodes(t.Context(), &hav1beta1.ListNodesRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Nodes)
	assert.Equal(t, int32(1), resp.ExpectedNodes)
}

func TestHAServer_ListNodes_ExpectedNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		nodes         []string
		expectedNodes int32
	}{
		{
			name:          "no peers configured defaults to 1",
			nodes:         nil,
			expectedNodes: 1,
		},
		{
			name:          "single peer configured",
			nodes:         []string{"node-1"},
			expectedNodes: 1,
		},
		{
			name:          "multiple peers configured",
			nodes:         []string{"node-1", "node-2", "node-3"},
			expectedNodes: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &Service{
				params: &models.HAParams{
					Enabled: true,
					Nodes:   tt.nodes,
				},
				memberlist: nil,
				rw:         sync.RWMutex{},
			}

			server := NewHAServer(service)

			resp, err := server.ListNodes(t.Context(), &hav1beta1.ListNodesRequest{})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Empty(t, resp.Nodes)
			assert.Equal(t, tt.expectedNodes, resp.ExpectedNodes)
		})
	}
}

func TestMemberlistStateToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		state          memberlist.NodeStateType
		expectedString string
	}{
		{
			name:           "StateAlive",
			state:          memberlist.StateAlive,
			expectedString: "alive",
		},
		{
			name:           "StateSuspect",
			state:          memberlist.StateSuspect,
			expectedString: "suspect",
		},
		{
			name:           "StateDead",
			state:          memberlist.StateDead,
			expectedString: "dead",
		},
		{
			name:           "StateLeft",
			state:          memberlist.StateLeft,
			expectedString: "left",
		},
		{
			name:           "Unknown state",
			state:          memberlist.NodeStateType(99),
			expectedString: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := memberlistStateToString(tt.state)
			assert.Equal(t, tt.expectedString, result)
		})
	}
}

func TestNewHAServer(t *testing.T) {
	t.Parallel()

	service := &Service{
		params: &models.HAParams{
			Enabled: true,
		},
	}

	server := NewHAServer(service)

	require.NotNil(t, server)
	assert.Equal(t, service, server.service)
}
