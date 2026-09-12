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

	"github.com/percona/pmm/managed/models"
)

type haServiceWithParams struct {
	params *models.HAParams
}

func (s haServiceWithParams) Params() *models.HAParams { return s.params }

func TestStateUpdaterVMAgentDeployment(t *testing.T) {
	testCases := []struct {
		name      string
		haEnabled bool
		agentID   string
		want      vmAgentDeployment
	}{
		{name: "standalone, client agent", agentID: "00000000-0000-4000-8000-000000000001"},
		{name: "standalone, server agent", agentID: models.PMMServerAgentID, want: vmAgentDeployment{isServerAgent: true}},
		{name: "HA, client agent", haEnabled: true, agentID: "00000000-0000-4000-8000-000000000001", want: vmAgentDeployment{haEnabled: true}},
		{name: "HA, server agent", haEnabled: true, agentID: models.PMMServerAgentID, want: vmAgentDeployment{haEnabled: true, isServerAgent: true}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := &StateUpdater{r: &Registry{haService: haServiceWithParams{&models.HAParams{Enabled: tc.haEnabled}}}}
			assert.Equal(t, tc.want, u.vmAgentDeployment(tc.agentID))
		})
	}
}
