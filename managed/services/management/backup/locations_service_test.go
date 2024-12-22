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

package backup

import (
	"context"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	backuppb "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestCreateBackupLocation(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedS3 := &mockAwsS3{}
	mockedS3.On("GetBucketLocation", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return("us-east-2", nil)
	svc := NewLocationsService(db, mockedS3)
	t.Run("add server config", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		})
		require.NoError(t, err)

		assert.NotEmpty(t, loc.LocationId)
	})

	t.Run("add client config", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		})
		require.NoError(t, err)

		assert.NotEmpty(t, loc.LocationId)
	})

	t.Run("add awsS3", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			S3Config: &backuppb.S3LocationConfig{
				Endpoint:   "https://awsS3.us-west-2.amazonaws.com/",
				AccessKey:  "access_key",
				SecretKey:  "secret_key",
				BucketName: "example_bucket",
			},
		})
		require.NoError(t, err)

		assert.NotEmpty(t, loc.LocationId)
	})

	t.Run("multiple configs", func(t *testing.T) {
		_, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
			S3Config: &backuppb.S3LocationConfig{
				Endpoint:   "https://awsS3.us-west-2.amazonaws.com/",
				AccessKey:  "access_key",
				SecretKey:  "secret_key",
				BucketName: "example_bucket",
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Only one config is allowed."), err)
	})
}

func TestListBackupLocations(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedS3 := &mockAwsS3{}
	mockedS3.On("GetBucketLocation", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return("us-east-2", nil)
	svc := NewLocationsService(db, mockedS3)

	req1 := &backuppb.AddLocationRequest{
		Name: gofakeit.Name(),
		FilesystemConfig: &backuppb.FilesystemLocationConfig{
			Path: "/tmp",
		},
	}
	res1, err := svc.AddLocation(ctx, req1)
	require.NoError(t, err)
	req2 := &backuppb.AddLocationRequest{
		Name: gofakeit.Name(),
		S3Config: &backuppb.S3LocationConfig{
			Endpoint:   "https://awsS3.us-west-2.amazonaws.com/",
			AccessKey:  "access_key",
			SecretKey:  "secret_key",
			BucketName: "example_bucket",
		},
	}
	res2, err := svc.AddLocation(ctx, req2)
	require.NoError(t, err)

	t.Run("list", func(t *testing.T) {
		res, err := svc.ListLocations(ctx, &backuppb.ListLocationsRequest{})
		require.NoError(t, err)

		checkLocation := func(id string, req *backuppb.AddLocationRequest) func() bool {
			return func() bool {
				for _, loc := range res.Locations {
					//nolint:nestif
					if loc.LocationId == id {
						if loc.Name != req.Name || loc.Description != req.Description {
							return false
						}
						if req.S3Config != nil {
							cfg := loc.Config.(*backuppb.Location_S3Config)
							if req.S3Config.Endpoint != cfg.S3Config.Endpoint ||
								req.S3Config.AccessKey != cfg.S3Config.AccessKey ||
								req.S3Config.SecretKey != cfg.S3Config.SecretKey ||
								req.S3Config.BucketName != cfg.S3Config.BucketName {
								return false
							}
						}
						if req.FilesystemConfig != nil {
							cfg := loc.Config.(*backuppb.Location_FilesystemConfig)
							if req.FilesystemConfig.Path != cfg.FilesystemConfig.Path {
								return false
							}
						}
						return true
					}
				}
				return false
			}
		}

		assert.Len(t, res.Locations, 2)

		assert.Condition(t, checkLocation(res1.LocationId, req1))
		assert.Condition(t, checkLocation(res2.LocationId, req2))
	})
}

func TestChangeBackupLocation(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedS3 := &mockAwsS3{}
	mockedS3.On("GetBucketLocation", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return("us-east-2", nil)
	svc := NewLocationsService(db, mockedS3)
	t.Run("update existing config", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, loc.LocationId)

		updateReq := &backuppb.ChangeLocationRequest{
			LocationId:  loc.LocationId,
			Name:        gofakeit.Name(),
			Description: gofakeit.Quote(),
			S3Config: &backuppb.S3LocationConfig{
				Endpoint:   "https://example.com",
				AccessKey:  "access_key",
				SecretKey:  "secret_key",
				BucketName: "example_bucket",
			},
		}
		_, err = svc.ChangeLocation(ctx, updateReq)
		require.NoError(t, err)

		updatedLocation, err := models.FindBackupLocationByID(db.Querier, loc.LocationId)
		require.NoError(t, err)
		assert.Equal(t, updateReq.Name, updatedLocation.Name)
		assert.Equal(t, updateReq.Description, updatedLocation.Description)
		assert.Nil(t, updatedLocation.FilesystemConfig)
		require.NotNil(t, updatedLocation.S3Config)
		assert.Equal(t, updateReq.S3Config.Endpoint, updatedLocation.S3Config.Endpoint)
		assert.Equal(t, updateReq.S3Config.AccessKey, updatedLocation.S3Config.AccessKey)
		assert.Equal(t, updateReq.S3Config.SecretKey, updatedLocation.S3Config.SecretKey)
		assert.Equal(t, updateReq.S3Config.BucketName, updatedLocation.S3Config.BucketName)
	})

	t.Run("update only name", func(t *testing.T) {
		addReq := &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		}
		loc, err := svc.AddLocation(ctx, addReq)
		require.NoError(t, err)
		require.NotEmpty(t, loc.LocationId)

		updateReq := &backuppb.ChangeLocationRequest{
			LocationId: loc.LocationId,
			Name:       gofakeit.Name(),
		}
		_, err = svc.ChangeLocation(ctx, updateReq)
		require.NoError(t, err)

		updatedLocation, err := models.FindBackupLocationByID(db.Querier, loc.LocationId)
		require.NoError(t, err)
		assert.Equal(t, updateReq.Name, updatedLocation.Name)
		require.NotNil(t, updatedLocation.FilesystemConfig)
		assert.Equal(t, addReq.FilesystemConfig.Path, updatedLocation.FilesystemConfig.Path)
	})

	t.Run("update to existing name", func(t *testing.T) {
		name := gofakeit.Name()
		_, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: name,
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		})
		require.NoError(t, err)

		loc2, err := svc.AddLocation(ctx, &backuppb.AddLocationRequest{
			Name: gofakeit.Name(),
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		})
		require.NoError(t, err)

		updateReq := &backuppb.ChangeLocationRequest{
			LocationId: loc2.LocationId,
			Name:       name,
			FilesystemConfig: &backuppb.FilesystemLocationConfig{
				Path: "/tmp",
			},
		}
		_, err = svc.ChangeLocation(ctx, updateReq)
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, fmt.Sprintf(`Location with name "%s" already exists.`, name)), err)
	})
}

func TestRemoveBackupLocation(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedS3 := &mockAwsS3{}
	svc := NewLocationsService(db, mockedS3)
	req := &backuppb.AddLocationRequest{
		Name: gofakeit.Name(),
		FilesystemConfig: &backuppb.FilesystemLocationConfig{
			Path: "/tmp",
		},
	}
	res1, err := svc.AddLocation(ctx, req)
	require.NoError(t, err)
	req.Name = gofakeit.Name()
	res2, err := svc.AddLocation(ctx, req)
	require.NoError(t, err)
	req.Name = gofakeit.Name()
	res3, err := svc.AddLocation(ctx, req)
	require.NoError(t, err)

	foundLocation := func(id string, locations []*backuppb.Location) bool {
		for _, loc := range locations {
			if loc.LocationId == id {
				return true
			}
		}
		return false
	}

	_, err = svc.RemoveLocation(ctx, &backuppb.RemoveLocationRequest{
		LocationId: res1.LocationId,
	})
	assert.NoError(t, err)

	_, err = svc.RemoveLocation(ctx, &backuppb.RemoveLocationRequest{
		LocationId: res3.LocationId,
	})
	assert.NoError(t, err)

	res, err := svc.ListLocations(ctx, &backuppb.ListLocationsRequest{})
	require.NoError(t, err)

	assert.False(t, foundLocation(res1.LocationId, res.Locations))
	assert.False(t, foundLocation(res3.LocationId, res.Locations))
	assert.True(t, foundLocation(res2.LocationId, res.Locations))

	// Try to remove non-existing location
	_, err = svc.RemoveLocation(ctx, &backuppb.RemoveLocationRequest{
		LocationId: "non-existing",
	})
	assert.ErrorIs(t, err, models.ErrNotFound)
}

func TestVerifyBackupLocationValidation(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedS3 := &mockAwsS3{}
	mockedS3.On("BucketExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(true, nil)

	svc := NewLocationsService(db, mockedS3)

	tableTests := []struct {
		name     string
		req      *backuppb.TestLocationConfigRequest
		errorMsg string
	}{
		{
			name: "client config - missing path",
			req: &backuppb.TestLocationConfigRequest{
				FilesystemConfig: &backuppb.FilesystemLocationConfig{
					Path: "",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = PMM client config path field is empty.",
		},
		{
			name:     "awsS3 config - missing config",
			req:      &backuppb.TestLocationConfigRequest{},
			errorMsg: "rpc error: code = InvalidArgument desc = Missing location config.",
		},
		{
			name: "awsS3 config - missing endpoint",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "",
					AccessKey:  "access_key",
					SecretKey:  "secret_key",
					BucketName: "example_bucket",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 endpoint field is empty.",
		},
		{
			name: "awsS3 config - missing access key",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "https://awsS3.us-west-2.amazonaws.com/",
					AccessKey:  "",
					SecretKey:  "secret_key",
					BucketName: "example_bucket",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 accessKey field is empty.",
		},
		{
			name: "awsS3 config - missing secret key",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "https://awsS3.us-west-2.amazonaws.com/",
					AccessKey:  "secret_key",
					SecretKey:  "",
					BucketName: "example_bucket",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 secretKey field is empty.",
		},
		{
			name: "awsS3 config - missing bucket name",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "https://awsS3.us-west-2.amazonaws.com/",
					AccessKey:  "secret_key",
					SecretKey:  "example_key",
					BucketName: "",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = S3 bucketName field is empty.",
		},
		{
			name: "awsS3 config - invalid endpoint",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "#invalidendpoint",
					AccessKey:  "secret_key",
					SecretKey:  "example_key",
					BucketName: "example_bucket",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = No host found in the Endpoint.",
		},
		{
			name: "awsS3 config - invalid endpoint, path is not allowed",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "https://awsS3.us-west-2.amazonaws.com/path",
					AccessKey:  "secret_key",
					SecretKey:  "example_key",
					BucketName: "example_bucket",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Path is not allowed for Endpoint.",
		},
		{
			name: "awsS3 config - invalid scheme",
			req: &backuppb.TestLocationConfigRequest{
				S3Config: &backuppb.S3LocationConfig{
					Endpoint:   "tcp://awsS3.us-west-2.amazonaws.com",
					AccessKey:  "secret_key",
					SecretKey:  "example_key",
					BucketName: "example_bucket",
				},
			},
			errorMsg: "rpc error: code = InvalidArgument desc = Invalid scheme 'tcp'",
		},
	}

	for _, test := range tableTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := svc.TestLocationConfig(ctx, test.req)
			if test.errorMsg != "" {
				assert.EqualError(t, err, test.errorMsg)
				return
			}
			assert.NoError(t, err)
		})
	}
}
