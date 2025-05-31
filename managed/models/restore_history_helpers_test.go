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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRestoreHistory(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	nodeID1 := "node_id_1"
	artifactID1, artifactID2 := "artifact_id_1", "artifact_id_2"
	serviceID1, serviceID2 := "service_id_1", "service_id_2"
	locationID1 := "location_id_1"

	prepareArtifactsAndService := func(q *reform.Querier) {
		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   nodeID1,
				NodeType: models.GenericNodeType,
				NodeName: "Node",
			},
			&models.Service{
				ServiceID:   serviceID1,
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service 1",
				NodeID:      nodeID1,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(777),
			},
			&models.Service{
				ServiceID:   serviceID2,
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service 2",
				NodeID:      nodeID1,
				Address:     pointer.ToString("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(777),
			},
			&models.BackupLocation{
				ID:          locationID1,
				Name:        "Location 1",
				Description: "Description for location 1",
				Type:        models.S3BackupLocationType,
				S3Config:    &models.S3LocationConfig{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			&models.Artifact{
				ID:         artifactID1,
				Name:       "artifact 1",
				Vendor:     "MySQL",
				LocationID: locationID1,
				ServiceID:  serviceID1,
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Snapshot,
				Status:     models.SuccessBackupStatus,
				Type:       models.OnDemandArtifactType,
				CreatedAt:  time.Now(),
			},
			&models.Artifact{
				ID:         artifactID2,
				Name:       "artifact 2",
				Vendor:     "MySQL",
				LocationID: locationID1,
				ServiceID:  serviceID2,
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Snapshot,
				Status:     models.SuccessBackupStatus,
				Type:       models.OnDemandArtifactType,
				CreatedAt:  time.Now(),
			},
		} {
			require.NoError(t, q.Insert(str))
		}
	}

	t.Run("create", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareArtifactsAndService(q)

		params := models.CreateRestoreHistoryItemParams{
			ArtifactID: artifactID1,
			ServiceID:  serviceID1,
			Status:     models.InProgressRestoreStatus,
		}

		i, err := models.CreateRestoreHistoryItem(q, params)
		require.NoError(t, err)
		assert.Equal(t, params.ArtifactID, i.ArtifactID)
		assert.Equal(t, params.ServiceID, i.ServiceID)
		assert.Equal(t, params.PITRTimestamp, i.PITRTimestamp)
		assert.Equal(t, params.Status, i.Status)
		assert.Less(t, time.Now().UTC().Unix()-i.StartedAt.Unix(), int64(5))
	})

	t.Run("change", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareArtifactsAndService(q)

		now := time.Now().Round(time.Second)

		params := models.CreateRestoreHistoryItemParams{
			ArtifactID:    artifactID1,
			ServiceID:     serviceID1,
			PITRTimestamp: &now,
			Status:        models.InProgressRestoreStatus,
		}

		i, err := models.CreateRestoreHistoryItem(q, params)
		require.NoError(t, err)
		assert.Equal(t, params.ArtifactID, i.ArtifactID)
		assert.Equal(t, params.ServiceID, i.ServiceID)
		assert.Equal(t, params.PITRTimestamp, i.PITRTimestamp)
		assert.Equal(t, params.Status, i.Status)
		assert.Less(t, time.Now().UTC().Unix()-i.StartedAt.Unix(), int64(5))

		i, err = models.ChangeRestoreHistoryItem(q, i.ID, models.ChangeRestoreHistoryItemParams{
			Status: models.ErrorRestoreStatus,
		})
		require.NoError(t, err)
		assert.Equal(t, params.ArtifactID, i.ArtifactID)
		assert.Equal(t, params.ServiceID, i.ServiceID)
		assert.WithinDuration(t, *params.PITRTimestamp, *i.PITRTimestamp, 0)
		assert.Equal(t, models.ErrorRestoreStatus, i.Status)
		assert.Less(t, time.Now().UTC().Unix()-i.StartedAt.Unix(), int64(5))
	})

	t.Run("list", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareArtifactsAndService(q)

		params1 := models.CreateRestoreHistoryItemParams{
			ArtifactID: artifactID1,
			ServiceID:  serviceID1,
			Status:     models.InProgressRestoreStatus,
		}
		params2 := models.CreateRestoreHistoryItemParams{
			ArtifactID: artifactID1,
			ServiceID:  serviceID2,
			Status:     models.SuccessRestoreStatus,
		}

		i1, err := models.CreateRestoreHistoryItem(q, params1)
		require.NoError(t, err)
		i2, err := models.CreateRestoreHistoryItem(q, params2)
		require.NoError(t, err)

		actual, err := models.FindRestoreHistoryItems(q, models.RestoreHistoryItemFilters{})
		require.NoError(t, err)

		found := func(id string) func() bool {
			return func() bool {
				for _, b := range actual {
					if b.ID == id {
						return true
					}
				}
				return false
			}
		}

		assert.Condition(t, found(i1.ID), "The first restore history item not found")
		assert.Condition(t, found(i2.ID), "The second restore history item not found")
	})

	t.Run("remove", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareArtifactsAndService(q)

		params := models.CreateRestoreHistoryItemParams{
			ArtifactID: artifactID1,
			ServiceID:  serviceID1,
			Status:     models.SuccessRestoreStatus,
		}
		i, err := models.CreateRestoreHistoryItem(q, params)
		require.NoError(t, err)

		err = models.RemoveRestoreHistoryItem(q, i.ID)
		require.NoError(t, err)

		artifacts, err := models.FindRestoreHistoryItems(q, models.RestoreHistoryItemFilters{})
		require.NoError(t, err)
		assert.Empty(t, artifacts)
	})
}

func TestRestoreHistoryValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		params   models.CreateRestoreHistoryItemParams
		errorMsg string
	}{
		{
			name: "artifact missing",
			params: models.CreateRestoreHistoryItemParams{
				ServiceID: "service_id",
				Status:    models.SuccessRestoreStatus,
			},
			errorMsg: "invalid argument: artifact_id shouldn't be empty",
		},
		{
			name: "service missing",
			params: models.CreateRestoreHistoryItemParams{
				ArtifactID: "artifact_id",
				Status:     models.SuccessRestoreStatus,
			},
			errorMsg: "invalid argument: service_id shouldn't be empty",
		},
		{
			name: "invalid status",
			params: models.CreateRestoreHistoryItemParams{
				ArtifactID: "artifact_id",
				ServiceID:  "service_id",
				Status:     models.RestoreStatus("invalid"),
			},
			errorMsg: "invalid argument: invalid status \"invalid\"",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c, err := models.CreateRestoreHistoryItem(nil, test.params)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
				assert.Nil(t, c)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}
