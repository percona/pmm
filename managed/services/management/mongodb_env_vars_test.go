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

package management

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

// Environment variable names are only meaningful for agents that pmm-agent starts as a separate
// process, because they are resolved into SetStateRequest.AgentProcess.env_variable_names. Among
// the MongoDB agents only mongodb_exporter is such a process; the QAN and RTA agents are built-in
// agents running inside pmm-agent. This test pins that only the exporter row stores the names, so
// we do not persist data no one reads.
func TestAddMongoDBStoresEnvVarNamesForExporterOnly(t *testing.T) {
	uuid.SetRand(&tests.IDReader{})
	t.Cleanup(func() { uuid.SetRand(nil) })

	ctx := logger.Set(context.Background(), t.Name())
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	cc := &mockConnectionChecker{}
	cc.Test(t)
	sib := &mockServiceInfoBroker{}
	sib.Test(t)
	state := &mockAgentsStateUpdater{}
	state.Test(t)
	ar := &mockAgentsRegistry{}
	ar.Test(t)
	vmdb := &mockPrometheusService{}
	vmdb.Test(t)
	vc := &mockVersionCache{}
	vc.Test(t)
	grafanaClient := &mockGrafanaClient{}
	grafanaClient.Test(t)
	vmClient := &mockVictoriaMetricsClient{}
	vmClient.Test(t)

	t.Cleanup(func() {
		cc.AssertExpectations(t)
		sib.AssertExpectations(t)
		state.AssertExpectations(t)
		ar.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		vc.AssertExpectations(t)
		grafanaClient.AssertExpectations(t)
		vmClient.AssertExpectations(t)
	})

	s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)

	state.On("RequestStateUpdate", ctx, models.PMMServerAgentID).Once()
	vc.On("RequestSoftwareVersionsUpdate").Once()

	envVarNames := []string{"KRB5_KTNAME", "KRB5_CONFIG"}

	resp, err := s.AddService(ctx, &managementv1.AddServiceRequest{
		Service: &managementv1.AddServiceRequest_Mongodb{
			Mongodb: &managementv1.AddMongoDBServiceParams{
				NodeId:                   models.PMMServerNodeID,
				ServiceName:              "mgmt-test-mongo-env-vars",
				Address:                  "127.0.0.1",
				Port:                     27017,
				PmmAgentId:               models.PMMServerAgentID,
				Username:                 "username",
				SkipConnectionCheck:      true,
				MetricsMode:              managementv1.MetricsMode_METRICS_MODE_PULL,
				EnvironmentVariableNames: envVarNames,
				QanMongodbProfiler:       true,
				QanMongodbMongolog:       true,
				RtaMongodbAgent:          true,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetMongodb())

	mongodb := resp.GetMongodb()
	require.NotNil(t, mongodb.GetMongodbExporter())
	require.NotNil(t, mongodb.GetQanMongodbProfiler())
	require.NotNil(t, mongodb.GetQanMongodbMongolog())
	require.NotNil(t, mongodb.GetRtaMongodbAgent())

	assert.Equal(t, envVarNames, mongodb.GetMongodbExporter().GetEnvironmentVariableNames())

	// Only the exporter row may carry the names; the built-in agents must have none.
	expected := map[models.AgentType][]string{
		models.MongoDBExporterType:         envVarNames,
		models.QANMongoDBProfilerAgentType: nil,
		models.QANMongoDBMongologAgentType: nil,
		models.RTAMongoDBAgentType:         nil,
	}

	agents, err := models.FindAgents(db.Querier, models.AgentFilters{
		ServiceID: mongodb.GetService().GetServiceId(),
	})
	require.NoError(t, err)
	require.Len(t, agents, len(expected))

	for _, agent := range agents {
		want, ok := expected[agent.AgentType]
		require.True(t, ok, "unexpected agent type %s", agent.AgentType)

		names, err := agent.GetEnvironmentVariableNames()
		require.NoError(t, err)
		assert.Equal(t, want, names, "agent type %s", agent.AgentType)
	}
}

// AddMongoDB must apply the same reserved-name rejection as AgentsService.AddMongoDBExporter:
// pmm-agent computes MONGODB_URI itself for mongodb_exporter, so a user-selected value here would
// silently never take effect.
func TestAddMongoDBRejectsReservedEnvVarName(t *testing.T) {
	uuid.SetRand(&tests.IDReader{})
	t.Cleanup(func() { uuid.SetRand(nil) })

	ctx := logger.Set(t.Context(), t.Name())
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	cc := &mockConnectionChecker{}
	cc.Test(t)
	sib := &mockServiceInfoBroker{}
	sib.Test(t)
	state := &mockAgentsStateUpdater{}
	state.Test(t)
	ar := &mockAgentsRegistry{}
	ar.Test(t)
	vmdb := &mockPrometheusService{}
	vmdb.Test(t)
	vc := &mockVersionCache{}
	vc.Test(t)
	grafanaClient := &mockGrafanaClient{}
	grafanaClient.Test(t)
	vmClient := &mockVictoriaMetricsClient{}
	vmClient.Test(t)

	t.Cleanup(func() {
		cc.AssertExpectations(t)
		sib.AssertExpectations(t)
		state.AssertExpectations(t)
		ar.AssertExpectations(t)
		vmdb.AssertExpectations(t)
		vc.AssertExpectations(t)
		grafanaClient.AssertExpectations(t)
		vmClient.AssertExpectations(t)
	})

	s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)

	_, err := s.AddService(ctx, &managementv1.AddServiceRequest{
		Service: &managementv1.AddServiceRequest_Mongodb{
			Mongodb: &managementv1.AddMongoDBServiceParams{
				NodeId:                   models.PMMServerNodeID,
				ServiceName:              "mgmt-test-mongo-env-vars-reserved",
				Address:                  "127.0.0.1",
				Port:                     27017,
				PmmAgentId:               models.PMMServerAgentID,
				Username:                 "username",
				SkipConnectionCheck:      true,
				MetricsMode:              managementv1.MetricsMode_METRICS_MODE_PULL,
				EnvironmentVariableNames: []string{"MONGODB_URI"},
			},
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	agents, err := models.FindAgents(db.Querier, models.AgentFilters{})
	require.NoError(t, err)
	for _, agent := range agents {
		assert.NotEqual(t, models.MongoDBExporterType, agent.AgentType, "rejected request must not persist an agent")
	}
}
