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
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestListPitrTimeranges(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedPbmPITRService := &mockPbmPITRService{}

	timelines := []backup.Timeline{
		{
			ReplicaSet: "rs0",
			Start:      uint32(time.Now().Unix()),
			End:        uint32(time.Now().Unix()),
		},
	}

	mockedPbmPITRService.On("ListPITRTimeranges", ctx, mock.Anything, mock.Anything, mock.Anything).Return(timelines, nil)
	artifactsService := NewArtifactsService(db, nil, mockedPbmPITRService)
	var locationID string

	params := models.CreateBackupLocationParams{
		Name:        gofakeit.Name(),
		Description: "",
	}
	params.S3Config = &models.S3LocationConfig{
		Endpoint:     "https://awsS3.us-west-2.amazonaws.com/",
		AccessKey:    "access_key",
		SecretKey:    "secret_key",
		BucketName:   "example_bucket",
		BucketRegion: "us-east-1",
	}
	loc, err := models.CreateBackupLocation(db.Querier, params)
	require.NoError(t, err)
	require.NotEmpty(t, loc.ID)

	locationID = loc.ID

	t.Run("successfully lists PITR time ranges", func(t *testing.T) {
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:        "test_artifact",
			Vendor:      "test_vendor",
			LocationID:  locationID,
			ServiceID:   "test_service",
			Mode:        models.PITR,
			DataModel:   models.LogicalDataModel,
			Status:      models.PendingBackupStatus,
			Compression: models.S2,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, artifact.ID)

		response, err := artifactsService.ListPitrTimeranges(ctx, &backuppb.ListPitrTimerangesRequest{
			ArtifactId: artifact.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Timeranges, 1)
	})

	t.Run("fails for invalid artifact ID", func(t *testing.T) {
		unknownID := "artifact_id/" + uuid.New().String()
		response, err := artifactsService.ListPitrTimeranges(ctx, &backuppb.ListPitrTimerangesRequest{
			ArtifactId: unknownID,
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Artifact with ID %q not found.", unknownID)), err)
		assert.Nil(t, response)
	})

	t.Run("fails for non-PITR artifact", func(t *testing.T) {
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:        "test_non_pitr_artifact",
			Vendor:      "test_vendor",
			LocationID:  locationID,
			ServiceID:   "test_service",
			Mode:        models.Snapshot,
			DataModel:   models.LogicalDataModel,
			Status:      models.PendingBackupStatus,
			Compression: models.ZSTD,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, artifact.ID)

		response, err := artifactsService.ListPitrTimeranges(ctx, &backuppb.ListPitrTimerangesRequest{
			ArtifactId: artifact.ID,
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Artifact is not a PITR artifact."), err)
		assert.Nil(t, response)
	})
	mock.AssertExpectationsForObjects(t, mockedPbmPITRService)
}

func TestConvertBackupCompression(t *testing.T) {
	tests := []struct {
		name        string
		compression models.BackupCompression
		expected    backuppb.BackupCompression
		shouldError bool
	}{
		{
			name:        "QuickLZ compression",
			compression: models.QuickLZ,
			expected:    backuppb.BackupCompression_QUICKLZ,
			shouldError: false,
		},
		{
			name:        "ZSTD compression",
			compression: models.ZSTD,
			expected:    backuppb.BackupCompression_ZSTD,
			shouldError: false,
		},
		{
			name:        "LZ4 compression",
			compression: models.LZ4,
			expected:    backuppb.BackupCompression_LZ4,
			shouldError: false,
		},
		{
			name:        "S2 compression",
			compression: models.S2,
			expected:    backuppb.BackupCompression_S2,
			shouldError: false,
		},
		{
			name:        "GZIP compression",
			compression: models.GZIP,
			expected:    backuppb.BackupCompression_GZIP,
			shouldError: false,
		},
		{
			name:        "Snappy compression",
			compression: models.Snappy,
			expected:    backuppb.BackupCompression_SNAPPY,
			shouldError: false,
		},
		{
			name:        "PGZIP compression",
			compression: models.PGZIP,
			expected:    backuppb.BackupCompression_PGZIP,
			shouldError: false,
		},
		{
			name:        "None compression",
			compression: models.None,
			expected:    backuppb.BackupCompression_NONE,
			shouldError: false,
		},
		{
			name:        "invalid compression",
			compression: "invalid",
			expected:    0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertBackupCompression(tt.compression)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Equal(t, backuppb.BackupCompression_BACKUP_COMPRESSION_INVALID, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestArtifactMetadataListToProto(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	params := models.CreateBackupLocationParams{
		Name:        gofakeit.Name(),
		Description: "",
	}
	params.S3Config = &models.S3LocationConfig{
		Endpoint:     "https://awsS3.us-west-2.amazonaws.com/",
		AccessKey:    "access_key",
		SecretKey:    "secret_key",
		BucketName:   "example_bucket",
		BucketRegion: "us-east-1",
	}
	loc, err := models.CreateBackupLocation(db.Querier, params)
	require.NoError(t, err)
	require.NotEmpty(t, loc.ID)

	artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:        "test_artifact",
		Vendor:      "test_vendor",
		LocationID:  loc.ID,
		ServiceID:   "test_service",
		Mode:        models.PITR,
		DataModel:   models.LogicalDataModel,
		Status:      models.PendingBackupStatus,
		Compression: models.S2,
	})
	assert.NoError(t, err)

	artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
		Metadata: &models.Metadata{
			FileList: []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}, {Name: "file3"}},
		},
	})
	require.NoError(t, err)

	restoreTo := time.Unix(123, 456)

	artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
		Metadata: &models.Metadata{
			FileList:       []models.File{{Name: "dir2", IsDirectory: true}, {Name: "file4"}, {Name: "file5"}, {Name: "file6"}},
			RestoreTo:      &restoreTo,
			BackupToolData: &models.BackupToolData{PbmMetadata: &models.PbmMetadata{Name: "backup tool data name"}},
		},
	})
	require.NoError(t, err)

	expected := []*backuppb.Metadata{
		{
			FileList: []*backuppb.File{
				{Name: "dir1", IsDirectory: true},
				{Name: "file1"},
				{Name: "file2"},
				{Name: "file3"},
			},
		},
		{
			FileList: []*backuppb.File{
				{Name: "dir2", IsDirectory: true},
				{Name: "file4"},
				{Name: "file5"},
				{Name: "file6"},
			},
			RestoreTo:          &timestamppb.Timestamp{Seconds: 123, Nanos: 456},
			BackupToolMetadata: &backuppb.Metadata_PbmMetadata{PbmMetadata: &backuppb.PbmMetadata{Name: "backup tool data name"}},
		},
	}

	actual := artifactMetadataListToProto(artifact)

	assert.Equal(t, expected, actual)
}
