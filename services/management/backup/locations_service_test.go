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

package backup

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit"
	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/utils/tests"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestCreateBackupLocation(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	svc := NewLocationsService(db)
	t.Run("add server config", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backupv1beta1.AddLocationRequest{
			Name: gofakeit.Name(),
			PmmServerConfig: &backupv1beta1.PMMServerLocationConfig{
				Path: "/tmp",
			},
		})
		assert.NoError(t, err)

		assert.NotEmpty(t, loc.LocationId)
	})

	t.Run("add client config", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backupv1beta1.AddLocationRequest{
			Name: gofakeit.Name(),
			PmmClientConfig: &backupv1beta1.PMMClientLocationConfig{
				Path: "/tmp",
			},
		})
		assert.NoError(t, err)

		assert.NotEmpty(t, loc.LocationId)
	})

	t.Run("add s3", func(t *testing.T) {
		loc, err := svc.AddLocation(ctx, &backupv1beta1.AddLocationRequest{
			Name: gofakeit.Name(),
			S3Config: &backupv1beta1.S3LocationConfig{
				Endpoint:  gofakeit.URL(),
				AccessKey: "access_key",
				SecretKey: "secret_key",
			},
		})
		assert.Nil(t, err)

		assert.NotEmpty(t, loc.LocationId)
	})

	t.Run("multiple configs", func(t *testing.T) {
		_, err := svc.AddLocation(ctx, &backupv1beta1.AddLocationRequest{
			Name: gofakeit.Name(),
			PmmClientConfig: &backupv1beta1.PMMClientLocationConfig{
				Path: "/tmp",
			},
			S3Config: &backupv1beta1.S3LocationConfig{
				Endpoint:  gofakeit.URL(),
				AccessKey: "access_key",
				SecretKey: "secret_key",
			},
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Only one config is allowed."), err)

	})
}

func TestListBackupLocations(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	svc := NewLocationsService(db)

	req1 := &backupv1beta1.AddLocationRequest{
		Name: gofakeit.Name(),
		PmmClientConfig: &backupv1beta1.PMMClientLocationConfig{
			Path: "/tmp",
		},
	}
	res1, err := svc.AddLocation(ctx, req1)
	require.Nil(t, err)
	req2 := &backupv1beta1.AddLocationRequest{
		Name: gofakeit.Name(),
		S3Config: &backupv1beta1.S3LocationConfig{
			Endpoint:  gofakeit.URL(),
			AccessKey: "access_key",
			SecretKey: "secret_key",
		},
	}
	res2, err := svc.AddLocation(ctx, req2)
	require.Nil(t, err)

	t.Run("list", func(t *testing.T) {
		res, err := svc.ListLocations(ctx, &backupv1beta1.ListLocationsRequest{})
		assert.Nil(t, err)

		checkLocation := func(id string, req *backupv1beta1.AddLocationRequest) func() bool {
			return func() bool {
				for _, loc := range res.Locations {
					if loc.LocationId == id {
						if loc.Name != req.Name || loc.Description != req.Description {
							return false
						}
						if req.S3Config != nil {
							cfg := loc.Config.(*backupv1beta1.Location_S3Config)
							if req.S3Config.Endpoint != cfg.S3Config.Endpoint ||
								req.S3Config.AccessKey != cfg.S3Config.AccessKey ||
								req.S3Config.SecretKey != cfg.S3Config.SecretKey {
								return false
							}

						}
						if req.PmmClientConfig != nil {
							cfg := loc.Config.(*backupv1beta1.Location_PmmClientConfig)
							if req.PmmClientConfig.Path != cfg.PmmClientConfig.Path {
								return false
							}
						}
						if req.PmmServerConfig != nil {
							cfg := loc.Config.(*backupv1beta1.Location_PmmServerConfig)
							if req.PmmServerConfig.Path != cfg.PmmServerConfig.Path {
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
