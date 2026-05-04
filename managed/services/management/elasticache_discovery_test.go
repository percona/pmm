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

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestElastiCacheDiscovery(t *testing.T) {
	t.Run("FindManagedServices", func(t *testing.T) {
		t.Run("NoServices", func(t *testing.T) {
			_ = logger.Set(t.Context(), t.Name())
			uuid.SetRand(&tests.IDReader{})
			t.Cleanup(func() { uuid.SetRand(nil) })

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			state := &mockAgentsStateUpdater{}
			state.Test(t)
			t.Cleanup(func() { state.AssertExpectations(t) })

			d := NewElastiCacheDiscovery(db, state)
			managed, err := d.findManagedServices()
			require.NoError(t, err)
			assert.Empty(t, managed)
		})

		t.Run("OnlyManagedServicesReturned", func(t *testing.T) {
			ctx := logger.Set(t.Context(), t.Name())
			_ = ctx
			uuid.SetRand(&tests.IDReader{})
			t.Cleanup(func() { uuid.SetRand(nil) })

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			state := &mockAgentsStateUpdater{}
			state.Test(t)
			t.Cleanup(func() { state.AssertExpectations(t) })

			// Create a node for the services.
			node, err := models.CreateNode(db.Querier, models.RemoteElastiCacheNodeType, &models.CreateNodeParams{
				NodeName: "test-node",
				Address:  "test-address",
				Region:   pointer.ToString("us-east-1"),
			})
			require.NoError(t, err)

			// Create a managed Valkey service (with managed_by label).
			_, err = models.AddNewService(db.Querier, models.ValkeyServiceType, &models.AddDBMSServiceParams{
				ServiceName: "managed-valkey",
				NodeID:      node.NodeID,
				Address:     pointer.ToString("managed.cache.amazonaws.com"),
				Port:        pointer.ToUint16(6379),
				CustomLabels: map[string]string{
					"managed_by": elasticacheManagedByLabel,
					"source":     "elasticache",
				},
			})
			require.NoError(t, err)

			// Create a non-managed Valkey service (no managed_by label).
			_, err = models.AddNewService(db.Querier, models.ValkeyServiceType, &models.AddDBMSServiceParams{
				ServiceName: "manual-valkey",
				NodeID:      node.NodeID,
				Address:     pointer.ToString("manual.cache.amazonaws.com"),
				Port:        pointer.ToUint16(6380),
			})
			require.NoError(t, err)

			d := NewElastiCacheDiscovery(db, state)
			managed, err := d.findManagedServices()
			require.NoError(t, err)
			require.Len(t, managed, 1)
			assert.Equal(t, "managed-valkey", managed[0].ServiceName)
		})
	})

	t.Run("AddAndRemoveInstance", func(t *testing.T) {
		ctx := logger.Set(t.Context(), t.Name())
		uuid.SetRand(&tests.IDReader{})
		t.Cleanup(func() { uuid.SetRand(nil) })

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		state := &mockAgentsStateUpdater{}
		state.Test(t)
		t.Cleanup(func() { state.AssertExpectations(t) })

		d := NewElastiCacheDiscovery(db, state)

		// Add an instance.
		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID)
		inst := discoveredInstance{
			Region:      "us-east-1",
			AZ:          "us-east-1a",
			ClusterID:   "my-cluster",
			NodeType:    "cache.r6g.large",
			Address:     "my-cluster.abc123.use1.cache.amazonaws.com",
			Port:        6379,
			Engine:      "valkey",
			TLS:         false,
			Environment: "production",
			Role:        "primary",
		}
		err := d.addInstance(ctx, inst)
		require.NoError(t, err)

		// Verify a managed service was created.
		managed, err := d.findManagedServices()
		require.NoError(t, err)
		require.Len(t, managed, 1)
		assert.Equal(t, "elasticache-my-cluster", managed[0].ServiceName)
		assert.Equal(t, "my-cluster.abc123.use1.cache.amazonaws.com", pointer.GetString(managed[0].Address))

		// Remove the service.
		err = d.removeService(ctx, managed[0])
		require.NoError(t, err)

		// Verify it's gone.
		managed, err = d.findManagedServices()
		require.NoError(t, err)
		assert.Empty(t, managed)
	})

	t.Run("AddReaderInstance", func(t *testing.T) {
		ctx := logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})
		t.Cleanup(func() { uuid.SetRand(nil) })

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		state := &mockAgentsStateUpdater{}
		state.Test(t)
		t.Cleanup(func() { state.AssertExpectations(t) })

		d := NewElastiCacheDiscovery(db, state)

		state.On("RequestStateUpdate", ctx, models.PMMServerAgentID)
		inst := discoveredInstance{
			Region:    "us-east-1",
			ClusterID: "my-cluster",
			Address:   "my-cluster-ro.abc123.use1.cache.amazonaws.com",
			Port:      6379,
			Engine:    "redis",
			Role:      "reader",
		}
		err := d.addInstance(ctx, inst)
		require.NoError(t, err)

		managed, err := d.findManagedServices()
		require.NoError(t, err)
		require.Len(t, managed, 1)
		assert.Equal(t, "elasticache-my-cluster-reader", managed[0].ServiceName)
	})
}
