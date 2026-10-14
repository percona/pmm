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

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
)

func changeNode(nodeID string, body nodes.ChangeNodeBody) (*nodes.ChangeNodeOK, error) {
	return client.Default.NodesService.ChangeNode(&nodes.ChangeNodeParams{
		NodeID:  nodeID,
		Body:    body,
		Context: pmmapitests.Context,
	})
}

func TestChangeNode(t *testing.T) {
	t.Parallel()

	t.Run("ChangesOnlyPassedFields", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddNode(t, &nodes.AddNodeBody{
			Generic: &nodes.AddNodeParamsBodyGeneric{
				NodeName:     pmmapitests.TestString(t, "node-for-change"),
				Address:      pmmapitests.TestString(t, "10.10.10.10"),
				NodeModel:    "m5",
				CustomLabels: map[string]string{"team": "db"},
			},
		}).Generic.NodeID

		res, err := changeNode(nodeID, nodes.ChangeNodeBody{
			Environment:  new("prod"),
			CustomLabels: &nodes.ChangeNodeParamsBodyCustomLabels{Values: map[string]string{"tier": "1"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "prod", res.Payload.Generic.Environment)
		assert.Equal(t, "m5", res.Payload.Generic.NodeModel)
		assert.Equal(t, map[string]string{"tier": "1"}, res.Payload.Generic.CustomLabels)

		got, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{NodeID: nodeID, Context: pmmapitests.Context})
		require.NoError(t, err)
		assert.Equal(t, "prod", got.Payload.Generic.Environment)
		assert.Equal(t, map[string]string{"tier": "1"}, got.Payload.Generic.CustomLabels)
	})

	// The REST gateway has to deliver an empty value as "clear", not as "no change".
	t.Run("EmptyValuesClearOverTheGateway", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddNode(t, &nodes.AddNodeBody{
			Generic: &nodes.AddNodeParamsBodyGeneric{
				NodeName:     pmmapitests.TestString(t, "node-for-clear"),
				Address:      pmmapitests.TestString(t, "10.10.10.11"),
				Region:       pmmapitests.TestString(t, "region"),
				CustomLabels: map[string]string{"team": "db"},
			},
		}).Generic.NodeID

		res, err := changeNode(nodeID, nodes.ChangeNodeBody{
			Region:       new(""),
			CustomLabels: &nodes.ChangeNodeParamsBodyCustomLabels{},
		})
		require.NoError(t, err)
		assert.Empty(t, res.Payload.Generic.Region)
		assert.Empty(t, res.Payload.Generic.CustomLabels)
	})

	t.Run("RemoteRDSRegionIsFixed", func(t *testing.T) {
		t.Parallel()

		rds := pmmapitests.AddRemoteRDSNode(t, pmmapitests.TestString(t, "rds-for-change"))

		_, err := changeNode(rds.NodeID, nodes.ChangeNodeBody{Region: new(pmmapitests.TestString(t, "other-region"))})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "The region of a Remote RDS Node can't be changed.")

		res, err := changeNode(rds.NodeID, nodes.ChangeNodeBody{Environment: new("prod")})
		require.NoError(t, err)
		assert.Equal(t, "prod", res.Payload.RemoteRDS.Environment)
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Parallel()

		nodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "node-for-dry-run")).NodeID

		res, err := changeNode(nodeID, nodes.ChangeNodeBody{Environment: new("prod"), DryRun: true})
		require.NoError(t, err)
		assert.Equal(t, "prod", res.Payload.Generic.Environment)

		got, err := client.Default.NodesService.GetNode(&nodes.GetNodeParams{NodeID: nodeID, Context: pmmapitests.Context})
		require.NoError(t, err)
		assert.Empty(t, got.Payload.Generic.Environment)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		_, err := changeNode("pmm-not-found", nodes.ChangeNodeBody{Environment: new("prod")})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Node with ID \"pmm-not-found\" not found.")
	})
}
