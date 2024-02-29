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

// Package agents contains business logic of working with pmm-agent.
package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/grafana"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

func TestRegistryAuthenticate(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*Registry, *grafana.Client, *reform.Querier, func(t *testing.T)) {
		t.Helper()

		c := grafana.NewClient("127.0.0.1:3000")
		vm, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Node",
			},
			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N1"),
				Version:      pointer.ToString("2.41.2"),
			},
		} {
			require.NoError(t, tx.Querier.Insert(str))
		}

		registry := NewRegistry(db, vm, c)
		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}

		return registry, c, tx.Querier, teardown
	}

	registry, grafanaClient, txQuerier, teardown := setup(t)
	defer teardown(t)

	t.Run("Basic authorization", func(t *testing.T) {
		agent, err := models.FindAgentByID(txQuerier, "A1")
		require.NoError(t, err)

		md := &agentpb.AgentConnectMetadata{
			ID:            agent.AgentID,
			Authorization: "Basic YWRtaW46YWRtaW4=",
			Version:       *agent.Version,
		}
		_, err = registry.authenticate(context.Background(), md, txQuerier)
		require.NoError(t, err)
	})

	t.Run("Basic wrong authorization", func(t *testing.T) {
		agent, err := models.FindAgentByID(txQuerier, "A1")
		require.NoError(t, err)

		md := &agentpb.AgentConnectMetadata{
			ID:            agent.AgentID,
			Authorization: "Basic wrong",
			Version:       *agent.Version,
		}
		_, err = registry.authenticate(context.Background(), md, txQuerier)
		require.Error(t, err)
	})

	t.Run("Bearer authorization", func(t *testing.T) {
		headersMD := metadata.New(map[string]string{
			"Authorization": "Basic YWRtaW46YWRtaW4=",
		})
		ctx := metadata.NewIncomingContext(context.Background(), headersMD)
		_, serviceToken, err := grafanaClient.CreateServiceAccount(ctx, "N1", true)
		require.NoError(t, err)
		defer grafanaClient.DeleteServiceAccount(ctx, "N1", true)

		agent, err := models.FindAgentByID(txQuerier, "A1")
		require.NoError(t, err)
		agentMD := &agentpb.AgentConnectMetadata{
			ID:            agent.AgentID,
			Authorization: fmt.Sprintf("Bearer %s", serviceToken),
			Version:       *agent.Version,
		}

		_, err = registry.authenticate(ctx, agentMD, txQuerier)
		require.NoError(t, err)
	})

	t.Run("Bearer wrong authorization", func(t *testing.T) {
		agent, err := models.FindAgentByID(txQuerier, "A1")
		require.NoError(t, err)
		agentMD := &agentpb.AgentConnectMetadata{
			ID:            agent.AgentID,
			Authorization: "Bearer wrong",
			Version:       *agent.Version,
		}

		headersMD := metadata.New(map[string]string{
			"Authorization": "Basic YWRtaW46YWRtaW4=",
		})
		ctx := metadata.NewIncomingContext(context.Background(), headersMD)
		_, err = registry.authenticate(ctx, agentMD, txQuerier)
		require.Error(t, err)
	})
}
