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
	"fmt"
	"net/url"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
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
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
			},
		}

		location, err := models.CreateBackupLocation(q, params)
		require.NoError(t, err)
		assert.Equal(t, models.FilesystemBackupLocationType, location.Type)
		assert.Equal(t, params.Name, location.Name)
		assert.Equal(t, params.Description, location.Description)
		assert.Equal(t, params.FilesystemConfig.Path, location.FilesystemConfig.Path)
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
					Endpoint:     "https://example.com/",
					AccessKey:    "access_key",
					SecretKey:    "secret_key",
					BucketName:   "example_bucket",
					BucketRegion: "us-east-2",
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
		assert.Equal(t, params.S3Config.BucketName, location.S3Config.BucketName)

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
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
				S3Config: &models.S3LocationConfig{
					Endpoint:     "https://example.com/",
					AccessKey:    "access_key",
					SecretKey:    "secret_key",
					BucketName:   "example_bucket",
					BucketRegion: "us-east-2",
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
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
			},
		}
		params2 := models.CreateBackupLocationParams{
			Name:        "some name2",
			Description: "some desc2",
			BackupLocationConfig: models.BackupLocationConfig{
				S3Config: &models.S3LocationConfig{
					Endpoint:     "https://example.com/",
					AccessKey:    "access_key",
					SecretKey:    "secret_key",
					BucketName:   "example_bucket",
					BucketRegion: "us-east-2",
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
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
			},
		}

		location, err := models.CreateBackupLocation(q, createParams)
		require.NoError(t, err)

		changeParams := models.ChangeBackupLocationParams{
			Name:        "some name2",
			Description: "",
			BackupLocationConfig: models.BackupLocationConfig{
				S3Config: &models.S3LocationConfig{
					Endpoint:     "https://example.com/",
					AccessKey:    "access_key",
					SecretKey:    "secret_key",
					BucketName:   "example_bucket",
					BucketRegion: "us-east-2",
				},
			},
		}

		updatedLoc, err := models.ChangeBackupLocation(q, location.ID, changeParams)
		require.NoError(t, err)
		assert.Equal(t, changeParams.Name, updatedLoc.Name)
		// We should change Description even if empty value is passed, otherwise user cannot clear the field.
		assert.Equal(t, changeParams.Description, updatedLoc.Description)
		assert.Equal(t, models.S3BackupLocationType, updatedLoc.Type)
		assert.Nil(t, updatedLoc.FilesystemConfig)
		assert.Equal(t, changeParams.S3Config, updatedLoc.S3Config)

		findLoc, err := models.FindBackupLocationByID(q, location.ID)
		require.NoError(t, err)

		assert.Equal(t, updatedLoc, findLoc)
	})

	t.Run("remove restrict", func(t *testing.T) {
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
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
			},
		}

		loc, err := models.CreateBackupLocation(q, params)
		require.NoError(t, err)

		err = models.RemoveBackupLocation(q, loc.ID, models.RemoveRestrict)
		require.NoError(t, err)

		locations, err := models.FindBackupLocations(q)
		require.NoError(t, err)
		assert.Empty(t, locations)
	})
	t.Run("remove cascade", func(t *testing.T) {
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
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
			},
		}

		loc, err := models.CreateBackupLocation(q, params)
		require.NoError(t, err)

		nodeID1, serviceID1 := "node_1", "service_1"
		node := &models.Node{
			NodeID:   nodeID1,
			NodeType: models.GenericNodeType,
			NodeName: "Node 1",
		}
		require.NoError(t, q.Insert(node))

		s := &models.Service{
			ServiceID:   serviceID1,
			ServiceType: models.MySQLServiceType,
			ServiceName: "Service 1",
			NodeID:      nodeID1,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(777),
		}
		require.NoError(t, q.Insert(s))

		artifact, err := models.CreateArtifact(q, models.CreateArtifactParams{
			Name:       "artifact",
			Vendor:     "mysql",
			LocationID: loc.ID,
			ServiceID:  serviceID1,
			DataModel:  models.PhysicalDataModel,
			Mode:       models.Snapshot,
			Status:     models.SuccessBackupStatus,
		})
		require.NoError(t, err)

		rhi, err := models.CreateRestoreHistoryItem(q, models.CreateRestoreHistoryItemParams{
			ArtifactID: artifact.ID,
			ServiceID:  serviceID1,
			Status:     models.SuccessRestoreStatus,
		})
		require.NoError(t, err)

		err = models.RemoveBackupLocation(q, loc.ID, models.RemoveRestrict)
		require.EqualError(t, err, fmt.Sprintf("rpc error: code = FailedPrecondition desc = "+
			"backup location with ID \"%s\" has artifacts.", loc.ID))

		err = models.RemoveBackupLocation(q, loc.ID, models.RemoveCascade)
		require.NoError(t, err)

		_, err = models.FindArtifactByID(q, artifact.ID)
		require.True(t, errors.Is(err, models.ErrNotFound))

		_, err = models.FindRestoreHistoryItemByID(q, rhi.ID)
		require.True(t, errors.Is(err, models.ErrNotFound))

		locations, err := models.FindBackupLocations(q)
		require.NoError(t, err)
		assert.Empty(t, locations)
	})
}

func TestCreateBackupLocationValidation(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	tableTests := []struct {
		name     string
		params   models.CreateBackupLocationParams
		errorMsg string
	}{
		{
			name: "normal client config",
			params: models.CreateBackupLocationParams{
				Name: "client-1",
				BackupLocationConfig: models.BackupLocationConfig{
					FilesystemConfig: &models.FilesystemLocationConfig{
						Path: "/tmp:dir_.-123",
					},
				},
			},
			errorMsg: "",
		},
		{
			name: "client config - missing path",
			params: models.CreateBackupLocationParams{
				Name: "client-2",
				BackupLocationConfig: models.BackupLocationConfig{
					FilesystemConfig: &models.FilesystemLocationConfig{
						Path: "",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = PMM client config path field is empty.",
		},
		{
			name: "client config - non-canonical",
			params: models.CreateBackupLocationParams{
				Name: "client-3",
				BackupLocationConfig: models.BackupLocationConfig{
					FilesystemConfig: &models.FilesystemLocationConfig{
						Path: "/some_directory/../../../root",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Specified folder in non-canonical format, canonical would be: \"/root\".",
		},
		{
			name: "client config - not absolute path",
			params: models.CreateBackupLocationParams{
				Name: "client-4",
				BackupLocationConfig: models.BackupLocationConfig{
					FilesystemConfig: &models.FilesystemLocationConfig{
						Path: "../../../my_directory",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Folder should be an absolute path (should contain leading slash).",
		},
		{
			name: "client config - not allowed symbols",
			params: models.CreateBackupLocationParams{
				Name: "client-5",
				BackupLocationConfig: models.BackupLocationConfig{
					FilesystemConfig: &models.FilesystemLocationConfig{
						Path: "/%my_directory",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Filesystem path can contain only dots, colons, slashes, letters, digits, underscores and dashes.",
		},
		{
			name: "normal s3 config",
			params: models.CreateBackupLocationParams{
				Name: "s3-1",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "https://s3.us-west-2.amazonaws.com/",
						AccessKey:    "access_key",
						SecretKey:    "secret_key",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "",
		},
		{
			name: "s3 config - missing endpoint",
			params: models.CreateBackupLocationParams{
				Name: "s3-2",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "",
						AccessKey:    "access_key",
						SecretKey:    "secret_key",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 endpoint field is empty.",
		},
		{
			name: "s3 config - missing access key",
			params: models.CreateBackupLocationParams{
				Name: "s3-3",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "https://s3.us-west-2.amazonaws.com/",
						AccessKey:    "",
						SecretKey:    "secret_key",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 accessKey field is empty.",
		},
		{
			name: "s3 config - missing secret key",
			params: models.CreateBackupLocationParams{
				Name: "s3-4",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "https://s3.us-west-2.amazonaws.com/",
						AccessKey:    "secret_key",
						SecretKey:    "",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 secretKey field is empty.",
		},
		{
			name: "s3 config - missing bucket name",
			params: models.CreateBackupLocationParams{
				Name: "s3-5",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "https://s3.us-west-2.amazonaws.com/",
						AccessKey:    "secret_key",
						SecretKey:    "example_key",
						BucketName:   "",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 bucketName field is empty.",
		},
		{
			name: "s3 config - invalid endpoint",
			params: models.CreateBackupLocationParams{
				Name: "s3-6",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "#invalidendpoint",
						AccessKey:    "secret_key",
						SecretKey:    "example_key",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = No host found in the Endpoint.",
		},
		{
			name: "s3 config - invalid endpoint, path is not allowed",
			params: models.CreateBackupLocationParams{
				Name: "s3-7",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "https://s3.us-west-2.amazonaws.com/path",
						AccessKey:    "secret_key",
						SecretKey:    "example_key",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Path is not allowed for Endpoint.",
		},
		{
			name: "s3 config - invalid scheme",
			params: models.CreateBackupLocationParams{
				Name: "s3-8",
				BackupLocationConfig: models.BackupLocationConfig{
					S3Config: &models.S3LocationConfig{
						Endpoint:     "tcp://s3.us-west-2.amazonaws.com",
						AccessKey:    "secret_key",
						SecretKey:    "example_key",
						BucketName:   "example_bucket",
						BucketRegion: "us-east-2",
					},
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Invalid scheme 'tcp'",
		},
	}

	for _, test := range tableTests {
		t.Run(test.name, func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			c, err := models.CreateBackupLocation(q, test.params)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, c)
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	tableTests := []struct {
		name     string
		endpoint string
		url      url.URL
		errorMsg string
	}{
		{
			name:     "HTTPS S3",
			endpoint: "https://s3.us-west-2.amazonaws.com",
			url: url.URL{
				Scheme: "https",
				Host:   "s3.us-west-2.amazonaws.com",
			},
		},
		{
			name:     "HTTP S3",
			endpoint: "http://s3.us-west-2.amazonaws.com",
			url: url.URL{
				Scheme: "http",
				Host:   "s3.us-west-2.amazonaws.com",
			},
		},
		{
			name:     "S3 without scheme",
			endpoint: "s3.us-west-2.amazonaws.com",
			url: url.URL{
				Scheme: "https",
				Host:   "s3.us-west-2.amazonaws.com",
			},
		},
		{
			name:     "Missing top level domain",
			endpoint: "1https://s3.us-west-2.amazonaws.com",
			errorMsg: "parse \"1https://s3.us-west-2.amazonaws.com\": first path segment in URL cannot contain colon",
		},
	}
	for _, test := range tableTests {
		t.Run(test.name, func(t *testing.T) {
			res, err := models.ParseEndpoint(test.endpoint)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.url, *res)
		})
	}
}
