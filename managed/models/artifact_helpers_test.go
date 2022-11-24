// Copyright (C) 2017 Percona LLC
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

//nolint:goconst
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

func TestArtifacts(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	nodeID1 := "node_id_1"
	serviceID1, serviceID2 := "service_id_1", "service_id_2"
	locationID1, locationID2 := "location_id_1", "location_id_2"

	prepareLocationsAndService := func(q *reform.Querier) {
		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   nodeID1,
				NodeType: models.GenericNodeType,
				NodeName: "Node 1",
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
			&models.BackupLocation{
				ID:          locationID2,
				Name:        "Location 2",
				Description: "Description for location 2",
				Type:        models.S3BackupLocationType,
				S3Config:    &models.S3LocationConfig{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
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
		prepareLocationsAndService(q)

		params := models.CreateArtifactParams{
			Name:       "backup_name",
			Vendor:     "MySQL",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
			Mode:       models.Snapshot,
		}

		a, err := models.CreateArtifact(q, params)
		require.NoError(t, err)
		assert.Equal(t, params.Name, a.Name)
		assert.Equal(t, params.Vendor, a.Vendor)
		assert.Equal(t, params.LocationID, a.LocationID)
		assert.Equal(t, params.ServiceID, a.ServiceID)
		assert.Equal(t, params.DataModel, a.DataModel)
		assert.Equal(t, params.Status, a.Status)
		assert.Less(t, time.Now().UTC().Unix()-a.CreatedAt.Unix(), int64(5))
	})

	t.Run("list", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		params1 := models.CreateArtifactParams{
			Name:       "backup_name_1",
			Vendor:     "MySQL",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
			Mode:       models.Snapshot,
		}
		params2 := models.CreateArtifactParams{
			Name:       "backup_name_2",
			Vendor:     "PostgreSQL",
			LocationID: locationID2,
			ServiceID:  serviceID2,
			DataModel:  models.LogicalDataModel,
			Status:     models.PausedBackupStatus,
			Mode:       models.Snapshot,
		}

		a1, err := models.CreateArtifact(q, params1)
		require.NoError(t, err)
		a2, err := models.CreateArtifact(q, params2)
		require.NoError(t, err)

		actual, err := models.FindArtifacts(q, models.ArtifactFilters{})
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

		assert.Condition(t, found(a1.ID), "The first artifact not found")
		assert.Condition(t, found(a2.ID), "The second artifact not found")
	})

	t.Run("remove", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		params := models.CreateArtifactParams{
			Name:       "backup_name",
			Vendor:     "MySQL",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
			Mode:       models.Snapshot,
		}

		b, err := models.CreateArtifact(q, params)
		require.NoError(t, err)

		err = models.DeleteArtifact(q, b.ID)
		require.NoError(t, err)

		artifacts, err := models.FindArtifacts(q, models.ArtifactFilters{})
		require.NoError(t, err)
		assert.Empty(t, artifacts)
	})
}

func TestArtifactValidation(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	testCases := []struct {
		name     string
		params   models.CreateArtifactParams
		errorMsg string
	}{
		{
			name: "name missing",
			params: models.CreateArtifactParams{
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
				Mode:       models.Snapshot,
			},
			errorMsg: "invalid argument: name shouldn't be empty",
		},
		{
			name: "vendor missing",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
				Mode:       models.Snapshot,
			},
			errorMsg: "invalid argument: vendor shouldn't be empty",
		},
		{
			name: "location missing",
			params: models.CreateArtifactParams{
				Name:      "backup_name",
				Vendor:    "MySQL",
				ServiceID: "service_id",
				DataModel: models.PhysicalDataModel,
				Status:    models.PendingBackupStatus,
				Mode:      models.Snapshot,
			},
			errorMsg: "invalid argument: location_id shouldn't be empty",
		},
		{
			name: "service missing",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
				Mode:       models.Snapshot,
			},
			errorMsg: "invalid argument: service_id shouldn't be empty",
		},
		{
			name: "empty backup mode",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				Mode:       "",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
			},
			errorMsg: "invalid argument: empty backup mode",
		},
		{
			name: "empty data model",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  "",
				Status:     models.PendingBackupStatus,
				Mode:       models.Snapshot,
			},
			errorMsg: "invalid argument: empty data model",
		},
		{
			name: "invalid data model",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.DataModel("invalid"),
				Status:     models.PendingBackupStatus,
				Mode:       models.Snapshot,
			},
			errorMsg: "invalid argument: invalid data model 'invalid'",
		},
		{
			name: "invalid status",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.BackupStatus("invalid"),
				Mode:       models.Snapshot,
			},
			errorMsg: "invalid argument: invalid status 'invalid'",
		},
		{
			name: "invalid mode",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
				Mode:       models.BackupMode("invalid"),
			},
			errorMsg: "invalid argument: invalid backup mode 'invalid'",
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.Rollback())
			})

			q := tx.Querier

			c, err := models.CreateArtifact(q, test.params)
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
