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
			u := &StateUpdater{r: &Registry{haService: haServiceStub{params: &models.HAParams{Enabled: tc.haEnabled}}}}
			assert.Equal(t, tc.want, u.vmAgentDeployment(tc.agentID))
		})
	}
}

func TestRTAAgentRunnable(t *testing.T) {
	mysql := &models.Service{ServiceType: models.MySQLServiceType}
	mongo := &models.Service{ServiceType: models.MongoDBServiceType}
	postgres := &models.Service{ServiceType: models.PostgreSQLServiceType}
	rtaMySQL := &models.Agent{AgentType: models.RTAMySQLAgentType}
	rtaMongo := &models.Agent{AgentType: models.RTAMongoDBAgentType}
	rtaPostgres := &models.Agent{AgentType: models.RTAPostgreSQLAgentType}
	exporter := &models.Agent{AgentType: models.MySQLdExporterType}

	for _, tc := range []struct {
		name    string
		row     *models.Agent
		service *models.Service
		version string
		want    bool
	}{
		{name: "MySQL RTA on 3.9.1 is withheld", row: rtaMySQL, service: mysql, version: "3.9.1"},
		{name: "MySQL RTA on 3.10.0 is sent", row: rtaMySQL, service: mysql, version: "3.10.0", want: true},
		{name: "MongoDB RTA on 3.6.0 is withheld", row: rtaMongo, service: mongo, version: "3.6.0"},
		{name: "MongoDB RTA on 3.9.1 is sent", row: rtaMongo, service: mongo, version: "3.9.1", want: true},
		{name: "PostgreSQL RTA on 3.9.1 is withheld", row: rtaPostgres, service: postgres, version: "3.9.1"},
		{name: "PostgreSQL RTA on 3.10.0 is sent", row: rtaPostgres, service: postgres, version: "3.10.0", want: true},
		{name: "other agents on 3.9.1 are sent", row: exporter, service: mysql, version: "3.9.1", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, rtaAgentRunnable(tc.row, tc.service, tc.version))
		})
	}
}
