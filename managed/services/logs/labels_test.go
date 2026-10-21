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

package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

var testServices = []serviceLabels{
	{cluster: "sharded", replicationSet: "rs1", nodeName: "rs101"},
	{cluster: "sharded", replicationSet: "rs1", nodeName: "rs102"},
	{cluster: "sharded", replicationSet: "rs2", nodeName: "rs201"},
	{cluster: "ps", environment: "dev", nodeName: "ps1"},
	{cluster: "ps", environment: "prod", nodeName: "ps2"},
	// A second service on ps1, in the same cluster.
	{cluster: "ps", environment: "dev", nodeName: "ps1"},
}

func TestLabelNodes(t *testing.T) {
	assert.Equal(t, []string{"rs101", "rs102", "rs201"}, labelNodes(testServices, []string{"sharded"}, nil, nil))
	assert.Equal(t, []string{"rs101", "rs102"}, labelNodes(testServices, []string{"sharded"}, nil, []string{"rs1"}),
		"each label narrows the services")
	assert.Empty(t, labelNodes(testServices, nil, []string{"dev"}, []string{"rs1"}))
	assert.Equal(t, []string{"rs101", "rs102", "rs201"}, labelNodes(testServices, nil, nil, []string{"rs1", "rs2"}),
		"values of one label add up")
	assert.Equal(t, []string{"ps1"}, labelNodes(testServices, []string{"ps"}, []string{"dev"}, nil))
	assert.Empty(t, labelNodes(testServices, []string{"missing"}, nil, nil))
}

func TestLabelFacets(t *testing.T) {
	clusters, environments, replicationSets := labelFacets(testServices, map[string]int64{"rs101": 5, "rs201": 1, "ps1": 7})
	assert.Equal(t, []*logsv1.Facet{{Value: "ps", Count: 7}, {Value: "sharded", Count: 6}}, clusters)
	assert.Equal(t, []*logsv1.Facet{{Value: "dev", Count: 7}}, environments, "prod's node has no lines")
	assert.Equal(t, []*logsv1.Facet{{Value: "rs1", Count: 5}, {Value: "rs2", Count: 1}}, replicationSets)
}
