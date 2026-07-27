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
	"fmt"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// internalNodePrefix mimics what the PMM HA Helm chart reports for its PostgreSQL cluster:
// the Nodes are named "<namespace>-<pod name>" by the PostgreSQL operator.
const internalNodePrefix = "pmm-pmm-ha-pg-db-"

func TestIsInternalNode(t *testing.T) {
	s := &ManagementService{internalNodePrefixes: []string{internalNodePrefix, "pmm-pmm-ha-ch-"}}

	for nodeName, expected := range map[string]bool{
		"pmm-pmm-ha-pg-db-instance1-qjjl-0": true,
		"pmm-pmm-ha-ch-0":                   true,
		"pmm-ha-0":                          false,
		"pmm-server":                        false,
		"":                                  false,
	} {
		assert.Equal(t, expected, s.isInternalNode(nodeName), nodeName)
	}

	t.Run("no prefixes configured", func(t *testing.T) {
		s := &ManagementService{}
		assert.False(t, s.isInternalNode(internalNodePrefix+"instance1-qjjl-0"))
	})
}

func TestAddServiceTarget(t *testing.T) {
	const (
		agentID = "00000000-0000-4000-8000-000000000005"
		address = "mysql.example.com"
	)

	for _, tc := range []struct {
		name            string
		req             *managementv1.AddServiceRequest
		expectedAgentID string
		expectedAddress string
	}{
		{
			name: "MySQL",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Mysql{
				Mysql: &managementv1.AddMySQLServiceParams{PmmAgentId: agentID, Address: address},
			}},
			expectedAgentID: agentID,
			expectedAddress: address,
		},
		{
			name: "MongoDB",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Mongodb{
				Mongodb: &managementv1.AddMongoDBServiceParams{PmmAgentId: agentID, Address: address},
			}},
			expectedAgentID: agentID,
			expectedAddress: address,
		},
		{
			name: "PostgreSQL",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Postgresql{
				Postgresql: &managementv1.AddPostgreSQLServiceParams{PmmAgentId: agentID, Address: address},
			}},
			expectedAgentID: agentID,
			expectedAddress: address,
		},
		{
			name: "ProxySQL",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Proxysql{
				Proxysql: &managementv1.AddProxySQLServiceParams{PmmAgentId: agentID, Address: address},
			}},
			expectedAgentID: agentID,
			expectedAddress: address,
		},
		{
			name: "Valkey",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Valkey{
				Valkey: &managementv1.AddValkeyServiceParams{PmmAgentId: agentID, Address: address},
			}},
			expectedAgentID: agentID,
			expectedAddress: address,
		},
		{
			name: "RDS",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Rds{
				Rds: &managementv1.AddRDSServiceParams{PmmAgentId: agentID, Address: address},
			}},
			expectedAgentID: agentID,
			expectedAddress: address,
		},
		{
			name: "External Services are scraped on the Node they run on",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_External{
				External: &managementv1.AddExternalServiceParams{RunsOnNodeId: "00000000-0000-4000-8000-000000000006"},
			}},
		},
		{
			name: "HAProxy Services are scraped on the Node they run on",
			req: &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Haproxy{
				Haproxy: &managementv1.AddHAProxyServiceParams{NodeId: "00000000-0000-4000-8000-000000000006"},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pmmAgentID, address := addServiceTarget(tc.req)
			assert.Equal(t, tc.expectedAgentID, pmmAgentID)
			assert.Equal(t, tc.expectedAddress, address)
		})
	}
}

func TestListNodesMarksInternalNodes(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: internalNodePrefix + "instance1-qjjl-0",
		Address:  "10.1.2.3",
	})
	require.NoError(t, err)

	ar := &mockAgentsRegistry{}
	ar.Test(t)
	ar.On("IsConnected", mock.Anything).Return(false)

	vmdb := &mockPrometheusService{}
	vmdb.Test(t)

	vmClient := &mockVictoriaMetricsClient{}
	vmClient.Test(t)
	vmClient.On("Query", ctx, mock.Anything, mock.Anything).Return(model.Vector{}, nil, nil)

	s := NewManagementService(db, ar, nil, nil, nil, vmdb, nil, nil, vmClient, []string{internalNodePrefix})

	res, err := s.ListNodes(ctx, &managementv1.ListNodesRequest{})
	require.NoError(t, err)

	isInternal := make(map[string]bool, len(res.Nodes))
	for _, n := range res.Nodes {
		isInternal[n.NodeName] = n.IsPmmInternalNode
	}
	assert.True(t, isInternal[node.NodeName], node.NodeName)
	assert.False(t, isInternal["pmm-server"])
}

func TestCheckNodeIsEligible(t *testing.T) {
	ctx := logger.Set(t.Context(), t.Name())

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: internalNodePrefix + "instance1-qjjl-0",
		Address:  "10.1.2.3",
	})
	require.NoError(t, err)
	agent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)

	s := NewManagementService(db, nil, nil, nil, nil, nil, nil, nil, nil, []string{internalNodePrefix})
	expectedErr := status.New(codes.FailedPrecondition, fmt.Sprintf(
		"Node '%s' is a part of the internal infrastructure of this PMM deployment and cannot monitor other services.", node.NodeName,
	))

	t.Run("a remote address on an internal Node is rejected", func(t *testing.T) {
		err := s.checkNodeIsEligible(ctx, agent.AgentID, "mysql.example.com")
		tests.AssertGRPCError(t, expectedErr, err)
	})

	t.Run("local addresses on an internal Node are allowed", func(t *testing.T) {
		for _, address := range []string{"", "localhost", "127.0.0.1", "::1"} {
			assert.NoError(t, s.checkNodeIsEligible(ctx, agent.AgentID, address), address)
		}
	})

	t.Run("a remote address on a regular Node is allowed", func(t *testing.T) {
		assert.NoError(t, s.checkNodeIsEligible(ctx, models.PMMServerAgentID, "mysql.example.com"))
	})

	t.Run("no prefixes configured", func(t *testing.T) {
		s := NewManagementService(db, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		assert.NoError(t, s.checkNodeIsEligible(ctx, agent.AgentID, "mysql.example.com"))
	})

	t.Run("AddService rejects an internal Node", func(t *testing.T) {
		res, err := s.AddService(ctx, &managementv1.AddServiceRequest{Service: &managementv1.AddServiceRequest_Mysql{
			Mysql: &managementv1.AddMySQLServiceParams{
				PmmAgentId:  agent.AgentID,
				ServiceName: "test-mysql",
				Address:     "mysql.example.com",
				Port:        3306,
			},
		}})
		assert.Nil(t, res)
		tests.AssertGRPCError(t, expectedErr, err)
	})

	t.Run("AddAzureDatabase rejects an internal Node", func(t *testing.T) {
		_, err := models.UpdateSettings(sqlDB, &models.ChangeSettingsParams{
			EnableAzurediscover: new(true),
		})
		require.NoError(t, err)

		res, err := s.AddAzureDatabase(ctx, &managementv1.AddAzureDatabaseRequest{
			PmmAgentId: agent.AgentID,
			InstanceId: "test-azure",
			Address:    "test.mysql.database.azure.com",
			Port:       3306,
		})
		assert.Nil(t, res)
		tests.AssertGRPCError(t, expectedErr, err)
	})
}
