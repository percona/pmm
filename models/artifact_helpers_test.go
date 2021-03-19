// pmm-managed
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

package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestArtifacts(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("create", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier

		params := models.CreateArtifactParams{
			Name:       "backup_name",
			Vendor:     "MySQL",
			LocationID: "location_id",
			ServiceID:  "service_id",
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
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

		params1 := models.CreateArtifactParams{
			Name:       "backup_name_1",
			Vendor:     "MySQL",
			LocationID: "location_id_1",
			ServiceID:  "service_id_1",
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
		}
		params2 := models.CreateArtifactParams{
			Name:       "backup_name_2",
			Vendor:     "PostgreSQL",
			LocationID: "location_id_2",
			ServiceID:  "service_id_2",
			DataModel:  models.LogicalDataModel,
			Status:     models.PausedBackupStatus,
		}

		a1, err := models.CreateArtifact(q, params1)
		require.NoError(t, err)
		a2, err := models.CreateArtifact(q, params2)
		require.NoError(t, err)

		actual, err := models.FindArtifacts(q)
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

		params := models.CreateArtifactParams{
			Name:       "backup_name",
			Vendor:     "MySQL",
			LocationID: "location_id",
			ServiceID:  "service_id",
			DataModel:  models.PhysicalDataModel,
			Status:     models.PendingBackupStatus,
		}

		b, err := models.CreateArtifact(q, params)
		require.NoError(t, err)

		err = models.RemoveArtifact(q, b.ID)
		require.NoError(t, err)

		artifacts, err := models.FindArtifacts(q)
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
			name: "normal params",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
			},
			errorMsg: "",
		},
		{
			name: "name missing",
			params: models.CreateArtifactParams{
				Vendor:     "MySQL",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
			},
			errorMsg: "name shouldn't be empty: invalid argument",
		},
		{
			name: "vendor missing",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				LocationID: "location_id",
				ServiceID:  "service_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
			},
			errorMsg: "vendor shouldn't be empty: invalid argument",
		},
		{
			name: "location missing",
			params: models.CreateArtifactParams{
				Name:      "backup_name",
				Vendor:    "MySQL",
				ServiceID: "service_id",
				DataModel: models.PhysicalDataModel,
				Status:    models.PendingBackupStatus,
			},
			errorMsg: "location_id shouldn't be empty: invalid argument",
		},
		{
			name: "service missing",
			params: models.CreateArtifactParams{
				Name:       "backup_name",
				Vendor:     "MySQL",
				LocationID: "location_id",
				DataModel:  models.PhysicalDataModel,
				Status:     models.PendingBackupStatus,
			},
			errorMsg: "service_id shouldn't be empty: invalid argument",
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
			},
			errorMsg: "invalid data model 'invalid': invalid argument",
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
			},
			errorMsg: "invalid status 'invalid': invalid argument",
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
