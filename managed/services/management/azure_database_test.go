// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
)

// TestAddAzureDatabaseRunsOnRequestedAgent covers the Agent an Azure Database is delegated to.
// The Agents are created under the requested pmm-agent, so it is the one which has to be told
// about them, otherwise it does not start monitoring until its next state update.
func TestAddAzureDatabaseRunsOnRequestedAgent(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
		EnableAzurediscover: new(true),
	})
	require.NoError(t, err)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "azure-delegate",
		Address:  "10.2.3.4",
	})
	require.NoError(t, err)
	agent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	state := &mockAgentsStateUpdater{}
	state.Test(t)
	state.On("RequestStateUpdate", ctx, agent.AgentID).Once()
	t.Cleanup(func() {
		state.AssertExpectations(t)
	})

	s := NewManagementService(db, nil, state, nil, nil, nil, nil, nil, nil, nil)

	res, err := s.AddAzureDatabase(ctx, &managementv1.AddAzureDatabaseRequest{
		PmmAgentId:            agent.AgentID,
		Region:                "westeurope",
		InstanceId:            "azure-mysql-instance",
		Address:               "test.mysql.database.azure.com",
		Port:                  3306,
		Username:              "azure-user",
		Type:                  managementv1.DiscoverAzureDatabaseType_DISCOVER_AZURE_DATABASE_TYPE_MYSQL,
		AzureDatabaseExporter: true,
		Qan:                   true,
		SkipConnectionCheck:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	agents, err := models.FindAgents(db.Querier, models.AgentFilters{PMMAgentID: agent.AgentID})
	require.NoError(t, err)

	agentTypes := make([]models.AgentType, 0, len(agents))
	for _, a := range agents {
		agentTypes = append(agentTypes, a.AgentType)
	}
	assert.ElementsMatch(t, []models.AgentType{
		models.AzureDatabaseExporterType,
		models.MySQLdExporterType,
		models.QANMySQLPerfSchemaAgentType,
	}, agentTypes)
}
