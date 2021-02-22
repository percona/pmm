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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestBackupLocations(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("create - pmm client", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		params := models.CreateBackupLocationParams{
			Name:        "some name",
			Description: "some desc",
			BackupLocationConfig: models.BackupLocationConfig{
				PMMClientConfig: &models.PMMClientLocationConfig{
					Path: "/tmp",
				},
			},
		}

		location, err := models.CreateBackupLocation(q, params)
		require.NoError(t, err)
		assert.Equal(t, models.PMMClientBackupLocationType, location.Type)
		assert.Equal(t, params.Name, location.Name)
		assert.Equal(t, params.Description, location.Description)
		assert.Equal(t, params.PMMClientConfig.Path, location.PMMClientConfig.Path)
		assert.NotEmpty(t, location.ID)
	})

	t.Run("create - s3", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		params := models.CreateBackupLocationParams{
			Name:        "some name",
			Description: "some desc",
			BackupLocationConfig: models.BackupLocationConfig{
				S3Config: &models.S3LocationConfig{
					Endpoint:  "https://example.com/bucket",
					AccessKey: "access_key",
					SecretKey: "secret_key",
				},
			},
		}

		location, err := models.CreateBackupLocation(q, params)
		require.NoError(t, err)
		assert.Equal(t, models.S3BackupLocationType, location.Type)
		assert.Equal(t, params.Name, location.Name)
		assert.Equal(t, params.Description, location.Description)
		assert.Equal(t, params.S3Config.Endpoint, location.S3Config.Endpoint)
		assert.Equal(t, params.S3Config.AccessKey, location.S3Config.AccessKey)
		assert.Equal(t, params.S3Config.SecretKey, location.S3Config.SecretKey)

		assert.NotEmpty(t, location.ID)
	})

	t.Run("create - two configs", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		params := models.CreateBackupLocationParams{
			Name:        "some name",
			Description: "some desc",
			BackupLocationConfig: models.BackupLocationConfig{
				PMMClientConfig: &models.PMMClientLocationConfig{
					Path: "/tmp",
				},
				S3Config: &models.S3LocationConfig{
					Endpoint:  "https://example.com/bucket",
					AccessKey: "access_key",
					SecretKey: "secret_key",
				},
			},
		}

		_, err = models.CreateBackupLocation(q, params)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Only one config is allowed."), err)
	})

	t.Run("list", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		params1 := models.CreateBackupLocationParams{
			Name:        "some name",
			Description: "some desc",
			BackupLocationConfig: models.BackupLocationConfig{
				PMMClientConfig: &models.PMMClientLocationConfig{
					Path: "/tmp",
				},
			},
		}
		params2 := models.CreateBackupLocationParams{
			Name:        "some name2",
			Description: "some desc2",
			BackupLocationConfig: models.BackupLocationConfig{
				S3Config: &models.S3LocationConfig{
					Endpoint:  "https://example.com/bucket",
					AccessKey: "access_key",
					SecretKey: "secret_key",
				},
			},
		}

		loc1, err := models.CreateBackupLocation(q, params1)
		require.NoError(t, err)
		loc2, err := models.CreateBackupLocation(q, params2)
		require.NoError(t, err)

		actual, err := models.FindBackupLocations(q)
		require.NoError(t, err)

		findLocID := func(id string) func() bool {
			return func() bool {
				for _, location := range actual {
					if location.ID == id {
						return true
					}
				}
				return false
			}
		}

		assert.Condition(t, findLocID(loc1.ID), "First location not found")
		assert.Condition(t, findLocID(loc2.ID), "Second location not found")
	})

	t.Run("update", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		createParams := models.CreateBackupLocationParams{
			Name:        "some name",
			Description: "some desc",
			BackupLocationConfig: models.BackupLocationConfig{
				PMMClientConfig: &models.PMMClientLocationConfig{
					Path: "/tmp",
				},
			},
		}

		location, err := models.CreateBackupLocation(q, createParams)
		require.NoError(t, err)

		changeParams := models.ChangeBackupLocationParams{
			Name:        "some name modified",
			Description: "",
			BackupLocationConfig: models.BackupLocationConfig{
				PMMServerConfig: &models.PMMServerLocationConfig{
					Path: "/tmp/nested",
				},
			},
		}

		updatedLoc, err := models.ChangeBackupLocation(q, location.ID, changeParams)
		require.NoError(t, err)
		assert.Equal(t, changeParams.Name, updatedLoc.Name)
		// empty description in request, we expect no change
		assert.Equal(t, createParams.Description, updatedLoc.Description)
		assert.Nil(t, updatedLoc.PMMClientConfig)
		assert.Equal(t, changeParams.PMMServerConfig.Path, updatedLoc.PMMServerConfig.Path)
		assert.Equal(t, updatedLoc.Type, models.PMMServerBackupLocationType)

		findLoc, err := models.FindBackupLocationByID(q, location.ID)

		require.NoError(t, err)

		assert.Equal(t, updatedLoc, findLoc)
	})
}

func TestBackupLocationValidation(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	tableTests := []struct {
		name     string
		location models.CreateBackupLocationParams
		errorMsg string
	}{
		{
			name: "normal client config",
			location: models.CreateBackupLocationParams{
				Name: "client-1",
				BackupLocationConfig: models.BackupLocationConfig{
					PMMClientConfig: &models.PMMClientLocationConfig{
						Path: "/tmp",
					},
				},
			},
			errorMsg: "",
		},
		{
			name: "client config - missing path",
			location: models.CreateBackupLocationParams{
				Name: "client-2",
				BackupLocationConfig: models.BackupLocationConfig{
					PMMClientConfig: &models.PMMClientLocationConfig{
						Path: "",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = PMM client config path field is empty.",
		},
		{
			name: "normal s3 config",
			location: models.CreateBackupLocationParams{
				Name: "s3-1",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:  "https://s3.us-west-2.amazonaws.com/mybucket",
						AccessKey: "access_key",
						SecretKey: "secret_key",
					},
				},
			},
			errorMsg: "",
		},
		{
			name: "s3 config - missing endpoint",
			location: models.CreateBackupLocationParams{
				Name: "s3-2",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:  "",
						AccessKey: "access_key",
						SecretKey: "secret_key",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 endpoint field is empty.",
		},
		{
			name: "s3 config - missing access key",
			location: models.CreateBackupLocationParams{
				Name: "s3-3",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:  "https://s3.us-west-2.amazonaws.com/mybucket",
						AccessKey: "",
						SecretKey: "secret_key",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 accessKey field is empty.",
		},
		{
			name: "s3 config - missing secret key",
			location: models.CreateBackupLocationParams{
				Name: "s3-4",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:  "https://s3.us-west-2.amazonaws.com/mybucket",
						AccessKey: "secret_key",
						SecretKey: "",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 secretKey field is empty.",
		},
	}

	for _, test := range tableTests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			c, err := models.CreateBackupLocation(q, test.location)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}
