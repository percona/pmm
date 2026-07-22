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
				Address:     new("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(777),
			},
			&models.Service{
				ServiceID:   serviceID2,
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service 2",
				NodeID:      nodeID1,
				Address:     new("127.0.0.1"),
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

	t.Run("create and update", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		createParams := models.CreateArtifactParams{
			Name:       "backup_name",
			Vendor:     "MySQL",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
			Mode:       models.Snapshot,
			Folder:     "artifact_folder",
		}

		a, err := models.CreateArtifact(q, createParams)
		require.NoError(t, err)
		require.NotNil(t, a)
		assert.Equal(t, createParams.Name, a.Name)
		assert.Equal(t, createParams.Vendor, a.Vendor)
		assert.Equal(t, createParams.LocationID, a.LocationID)
		assert.Equal(t, createParams.ServiceID, a.ServiceID)
		assert.Equal(t, createParams.DataModel, a.DataModel)
		assert.Equal(t, createParams.Status, a.Status)
		assert.Equal(t, createParams.Folder, a.Folder)
		assert.Equal(t, models.OnDemandArtifactType, a.Type)
		assert.Less(t, time.Now().UTC().Unix()-a.CreatedAt.Unix(), int64(5))

		updateParams := models.UpdateArtifactParams{
			Status:           models.SuccessBackupStatus.Pointer(),
			ScheduleID:       new("schedule_id"),
			ServiceID:        &serviceID2,
			IsShardedCluster: true,
		}

		a, err = models.UpdateArtifact(q, a.ID, updateParams)
		require.NoError(t, err)
		require.NotNil(t, a)
		assert.Equal(t, *updateParams.Status, a.Status)
		assert.Equal(t, *updateParams.ScheduleID, a.ScheduleID)
		assert.Equal(t, *updateParams.ServiceID, a.ServiceID)
		assert.Equal(t, updateParams.IsShardedCluster, a.IsShardedCluster)
		assert.Less(t, time.Now().UTC().Unix()-a.UpdatedAt.Unix(), int64(5))

		t.Run("scheduled type", func(t *testing.T) {
			createParams.Name = "scheduled_backup"
			createParams.ScheduleID = "some_schedule"
			sa, err := models.CreateArtifact(q, createParams)
			require.NoError(t, err)
			assert.Equal(t, models.ScheduledArtifactType, sa.Type)
			assert.Equal(t, "some_schedule", sa.ScheduleID)
		})
	})

	t.Run("unique name", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		params := models.CreateArtifactParams{
			Name:       "unique_name",
			Vendor:     "MySQL",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
			Mode:       models.Snapshot,
		}

		_, err = models.CreateArtifact(q, params)
		require.NoError(t, err)

		_, err = models.CreateArtifact(q, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `Artifact with name "unique_name" already exists.`)
	})

	t.Run("find by ID and name", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		a, err := models.CreateArtifact(q, models.CreateArtifactParams{
			Name:       "find_me",
			Vendor:     "MySQL",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
			Mode:       models.Snapshot,
		})
		require.NoError(t, err)

		// Find by ID
		found, err := models.FindArtifactByID(q, a.ID)
		require.NoError(t, err)
		assert.Equal(t, a.Name, found.Name)

		_, err = models.FindArtifactByID(q, "non-existent")
		require.ErrorIs(t, err, models.ErrNotFound)

		// Find by Name
		found, err = models.FindArtifactByName(q, "find_me")
		require.NoError(t, err)
		assert.Equal(t, a.ID, found.ID)

		_, err = models.FindArtifactByName(q, "unknown")
		assert.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("find by IDs", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		a1, _ := models.CreateArtifact(q, models.CreateArtifactParams{Name: "a1", Vendor: "V", LocationID: locationID1, ServiceID: serviceID1, DataModel: models.PhysicalDataModel, Status: models.PendingBackupStatus, Mode: models.Snapshot})
		a2, _ := models.CreateArtifact(q, models.CreateArtifactParams{Name: "a2", Vendor: "V", LocationID: locationID1, ServiceID: serviceID1, DataModel: models.PhysicalDataModel, Status: models.PendingBackupStatus, Mode: models.Snapshot})

		res, err := models.FindArtifactsByIDs(q, []string{a1.ID, a2.ID, "unknown"})
		require.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Contains(t, res, a1.ID)
		assert.Contains(t, res, a2.ID)

		emptyRes, err := models.FindArtifactsByIDs(q, []string{})
		require.NoError(t, err)
		assert.Empty(t, emptyRes)
	})

	t.Run("list and filters", func(t *testing.T) {
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

		params3 := models.CreateArtifactParams{
			Name:       "backup_name_3",
			Vendor:     "mongodb",
			LocationID: locationID2,
			ServiceID:  serviceID2,
			DataModel:  models.LogicalDataModel,
			Status:     models.SuccessBackupStatus,
			Mode:       models.Snapshot,
			Folder:     "some_folder",
		}

		a1, err := models.CreateArtifact(q, params1)
		require.NoError(t, err)
		a2, err := models.CreateArtifact(q, params2)
		require.NoError(t, err)
		a3, err := models.CreateArtifact(q, params3)
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

		// Check artifacts can be found by folder.
		actual2, err := models.FindArtifacts(q, models.ArtifactFilters{Folder: &a3.Folder})
		require.NoError(t, err)
		assert.Equal(t, []*models.Artifact{a3}, actual2)

		// Check filters by status and service.
		actual3, err := models.FindArtifacts(q, models.ArtifactFilters{Status: models.PausedBackupStatus, ServiceID: serviceID2})
		require.NoError(t, err)
		assert.Len(t, actual3, 1)
		assert.Equal(t, a2.ID, actual3[0].ID)

		// Check non-existent location error.
		_, err = models.FindArtifacts(q, models.ArtifactFilters{LocationID: "invalid_loc"})
		require.Error(t, err)

		// Check sorting (latest first).
		all, err := models.FindArtifacts(q, models.ArtifactFilters{})
		require.NoError(t, err)
		require.Len(t, all, 3)
		assert.Equal(t, a3.ID, all[0].ID)
		assert.Equal(t, a2.ID, all[1].ID)
		assert.Equal(t, a1.ID, all[2].ID)
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

	t.Run("MetadataRemoveFirstN", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		prepareLocationsAndService(q)

		params := models.CreateArtifactParams{
			Name:       "backup_name",
			Vendor:     "MongoDB",
			LocationID: locationID1,
			ServiceID:  serviceID1,
			DataModel:  models.LogicalDataModel,
			Status:     models.SuccessBackupStatus,
			Mode:       models.PITR,
		}

		a, err := models.CreateArtifact(q, params)
		require.NotNil(t, a)
		require.NoError(t, err)

		a, err = models.UpdateArtifact(q, a.ID, models.UpdateArtifactParams{Metadata: &models.Metadata{FileList: []models.File{{Name: "file1"}}}})
		require.NoError(t, err)

		a, err = models.UpdateArtifact(q, a.ID, models.UpdateArtifactParams{Metadata: &models.Metadata{FileList: []models.File{{Name: "file2"}}}})
		require.NoError(t, err)

		a, err = models.UpdateArtifact(q, a.ID, models.UpdateArtifactParams{Metadata: &models.Metadata{FileList: []models.File{{Name: "file3"}}}})
		require.NoError(t, err)

		a, err = models.UpdateArtifact(q, a.ID, models.UpdateArtifactParams{Metadata: &models.Metadata{FileList: []models.File{{Name: "file4"}}}})
		require.NoError(t, err)

		err = a.MetadataRemoveFirstN(q, 0)
		require.NoError(t, err)
		assert.Len(t, a.MetadataList, 4)

		err = a.MetadataRemoveFirstN(q, 3)
		require.NoError(t, err)
		assert.Len(t, a.MetadataList, 1)

		err = a.MetadataRemoveFirstN(q, 10)
		require.NoError(t, err)
		assert.Empty(t, a.MetadataList)
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
		t.Run(test.name, func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.Rollback())
			})

			q := tx.Querier

			c, err := models.CreateArtifact(q, test.params)
			if test.errorMsg != "" {
				require.EqualError(t, err, test.errorMsg)
				assert.Nil(t, c)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}

func TestIsArtifactFinalStatus(t *testing.T) {
	testCases := []struct {
		status   models.BackupStatus
		expected bool
	}{
		{models.SuccessBackupStatus, true},
		{models.ErrorBackupStatus, true},
		{models.FailedToDeleteBackupStatus, true},
		{models.PendingBackupStatus, false},
		{models.PausedBackupStatus, false},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.expected, models.IsArtifactFinalStatus(tc.status), "Status: %s", tc.status)
	}
}
