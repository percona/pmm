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
	"database/sql"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	endpoint     = "https://s3.us-west-2.amazonaws.com/"
	accessKey    = "access_key"
	secretKey    = "secret_key"
	bucketName   = "example_bucket"
	bucketRegion = "us-east-2"
)

func TestDeleteArtifact(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedPbmPITRService := &mockPbmPITRService{}
	removalService := NewRemovalService(db, mockedPbmPITRService)

	agent, _ := setup(t, db.Querier, models.MySQLServiceType, "test-service")

	s3Config := &models.S3LocationConfig{
		Endpoint:     endpoint,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		BucketName:   bucketName,
		BucketRegion: bucketRegion,
	}

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: s3Config,
		},
	})
	require.NoError(t, err)

	createArtifact := func(status models.BackupStatus) *models.Artifact {
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "artifact_name",
			Vendor:     "MySQL",
			LocationID: locationRes.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.PhysicalDataModel,
			Mode:       models.Snapshot,
			Status:     status,
		})
		require.NoError(t, err)
		return artifact
	}

	mockedStorage := &MockStorage{}

	t.Run("artifact not in final status", func(t *testing.T) {
		artifact := createArtifact(models.PendingBackupStatus)
		t.Cleanup(func() {
			err := models.DeleteArtifact(db.Querier, artifact.ID)
			require.NoError(t, err)
		})

		err := removalService.DeleteArtifact(mockedStorage, artifact.ID, false)
		require.ErrorIs(t, err, ErrIncorrectArtifactStatus)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.PendingBackupStatus, artifact.Status)
	})

	t.Run("failed to remove restore history sets artifact error status", func(t *testing.T) {
		artifact := createArtifact(models.SuccessBackupStatus)
		t.Cleanup(func() {
			err := models.DeleteArtifact(db.Querier, artifact.ID)
			require.NoError(t, err)
		})

		ri, err := models.CreateRestoreHistoryItem(db.Querier, models.CreateRestoreHistoryItemParams{
			ArtifactID: artifact.ID,
			ServiceID:  *agent.ServiceID,
			Status:     models.SuccessRestoreStatus,
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
			require.NoError(t, err)

			err = models.RemoveRestoreHistoryItem(tx.Querier, ri.ID)
			require.NoError(t, err)

			err = tx.Commit()
			assert.NoError(t, err)
		})

		time.Sleep(time.Second)

		err = removalService.DeleteArtifact(mockedStorage, artifact.ID, false)
		require.Error(t, err)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.FailedToDeleteBackupStatus, artifact.Status)
	})

	t.Run("error during removing files", func(t *testing.T) {
		artifact := createArtifact(models.SuccessBackupStatus)
		t.Cleanup(func() {
			err := models.DeleteArtifact(db.Querier, artifact.ID)
			require.NoError(t, err)
		})

		someError := errors.New("some error")
		mockedStorage.On("RemoveRecursive", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, artifact.Name+"/").
			Return(someError).Once()

		err := removalService.DeleteArtifact(mockedStorage, artifact.ID, true)
		// No error because removing files running in goroutine.
		require.NoError(t, err)

		// Removing files running in goroutine, need to wait some time.
		time.Sleep(time.Second * 1)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.FailedToDeleteBackupStatus, artifact.Status)
	})

	t.Run("successful delete snapshot", func(t *testing.T) {
		artifact := createArtifact(models.SuccessBackupStatus)

		mockedStorage.On("RemoveRecursive", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, artifact.Name+"/").
			Return(nil).Once()

		err := removalService.DeleteArtifact(mockedStorage, artifact.ID, true)
		assert.NoError(t, err)

		// Removing files running in goroutine, need to wait some time.
		time.Sleep(time.Second * 3)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		assert.Nil(t, artifact)
		assert.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("successful delete pitr", func(t *testing.T) {
		agent, _ := setup(t, db.Querier, models.MongoDBServiceType, "test-service2")

		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "artifact_name",
			Vendor:     "mongodb",
			LocationID: locationRes.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.LogicalDataModel,
			Mode:       models.PITR,
			Status:     models.SuccessBackupStatus,
			Folder:     "artifact_folder",
		})
		require.NoError(t, err)

		artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
			Metadata: &models.Metadata{
				FileList: []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}, {Name: "file3"}},
			},
		})
		require.NoError(t, err)

		chunksRet := []*oplogChunk{
			{FName: "chunk1"},
			{FName: "chunk2"},
			{FName: "chunk3"},
		}

		mockedStorage.On("RemoveRecursive", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/dir1/").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/file1").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/file2").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/file3").
			Return(nil).Once()

		mockedPbmPITRService.On("GetPITRFiles", mock.Anything, mock.Anything, locationRes, artifact, mock.Anything).Return(chunksRet, nil).Once()

		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "chunk1").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "chunk2").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "chunk3").
			Return(nil).Once()

		err = removalService.DeleteArtifact(mockedStorage, artifact.ID, true)
		assert.NoError(t, err)

		// Removing files running in goroutine, need to wait some time.
		time.Sleep(time.Second * 3)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		assert.Nil(t, artifact)
		assert.ErrorIs(t, err, models.ErrNotFound)
	})

	mockedPbmPITRService.AssertExpectations(t)
	mockedStorage.AssertExpectations(t)
}

func TestTrimPITRArtifact(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedPbmPITRService := &mockPbmPITRService{}
	removalService := NewRemovalService(db, mockedPbmPITRService)
	mockedStorage := &MockStorage{}

	agent, _ := setup(t, db.Querier, models.MongoDBServiceType, "test-service2")

	s3Config := &models.S3LocationConfig{
		Endpoint:     endpoint,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		BucketName:   bucketName,
		BucketRegion: bucketRegion,
	}

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: s3Config,
		},
	})
	require.NoError(t, err)

	artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:       "artifact_name",
		Vendor:     "MongoDB",
		LocationID: locationRes.ID,
		ServiceID:  *agent.ServiceID,
		DataModel:  models.LogicalDataModel,
		Mode:       models.PITR,
		Status:     models.PendingBackupStatus,
		Folder:     "artifact_folder",
	})
	require.NoError(t, err)

	artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
		Metadata: &models.Metadata{
			FileList: []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}, {Name: "file3"}},
		},
	})
	require.NoError(t, err)

	restoreTo := time.Unix(123, 456)

	artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
		Metadata: &models.Metadata{
			FileList:  []models.File{{Name: "dir2", IsDirectory: true}, {Name: "file4"}, {Name: "file5"}, {Name: "file6"}},
			RestoreTo: &restoreTo,
		},
	})
	require.NoError(t, err)

	artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
		Metadata: &models.Metadata{
			FileList: []models.File{{Name: "dir3", IsDirectory: true}, {Name: "file7"}, {Name: "file8"}, {Name: "file9"}},
		},
	})
	require.NoError(t, err)

	t.Run("artifact not in final status", func(t *testing.T) {
		err := removalService.TrimPITRArtifact(mockedStorage, artifact.ID, 1)
		require.ErrorIs(t, err, ErrIncorrectArtifactStatus)

		time.Sleep(time.Second * 2)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.PendingBackupStatus, artifact.Status)
		assert.Len(t, artifact.MetadataList, 3)
	})

	t.Run("error during removing files sets artifact status", func(t *testing.T) {
		artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{Status: models.SuccessBackupStatus.Pointer()})
		require.NoError(t, err)

		mockedStorage.On("RemoveRecursive", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/dir1/").
			Return(errors.New("some error")).Once()

		err := removalService.TrimPITRArtifact(mockedStorage, artifact.ID, 1)
		require.NoError(t, err)

		time.Sleep(time.Second * 2)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.SuccessBackupStatus, artifact.Status)
		assert.Len(t, artifact.MetadataList, 3)
	})

	t.Run("successful", func(t *testing.T) {
		chunksRet := []*oplogChunk{
			{FName: "chunk1"},
			{FName: "chunk2"},
			{FName: "chunk3"},
		}

		mockedStorage.On("RemoveRecursive", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/dir1/").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/file1").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/file2").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "artifact_folder/file3").
			Return(nil).Once()

		mockedPbmPITRService.On("GetPITRFiles", mock.Anything, mock.Anything, locationRes, mock.Anything, artifact.MetadataList[1].RestoreTo).Return(chunksRet, nil).Once()

		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "chunk1").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "chunk2").
			Return(nil).Once()
		mockedStorage.On("Remove", mock.Anything, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, "chunk3").
			Return(nil).Once()

		err := removalService.TrimPITRArtifact(mockedStorage, artifact.ID, 1)
		require.NoError(t, err)

		time.Sleep(time.Second * 2)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.SuccessBackupStatus, artifact.Status)
		assert.Len(t, artifact.MetadataList, 2)
	})

	mockedStorage.AssertExpectations(t)
	mockedPbmPITRService.AssertExpectations(t)
}

func TestLockArtifact(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	agent, _ := setup(t, db.Querier, models.MongoDBServiceType, "test-service3")

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			FilesystemConfig: &models.FilesystemLocationConfig{Path: "/"},
		},
	})
	require.NoError(t, err)

	artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:       "artifact_name",
		Vendor:     "MongoDB",
		LocationID: locationRes.ID,
		ServiceID:  *agent.ServiceID,
		DataModel:  models.LogicalDataModel,
		Mode:       models.PITR,
		Status:     models.PendingBackupStatus,
		Folder:     "artifact_folder",
	})
	require.NoError(t, err)

	removalService := NewRemovalService(db, nil)

	t.Run("wrong locking status", func(t *testing.T) {
		res, oldStatus, err := removalService.lockArtifact(artifact.ID, models.FailedToDeleteBackupStatus)
		assert.Nil(t, res)
		assert.Empty(t, oldStatus)
		assert.ErrorIs(t, err, ErrIncorrectArtifactStatus)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.PendingBackupStatus, artifact.Status)
	})

	t.Run("artifact not in final status", func(t *testing.T) {
		res, oldStatus, err := removalService.lockArtifact(artifact.ID, models.DeletingBackupStatus)
		assert.Nil(t, res)
		assert.Empty(t, oldStatus)
		assert.ErrorIs(t, err, ErrIncorrectArtifactStatus)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.PendingBackupStatus, artifact.Status)
	})

	t.Run("restore in progress", func(t *testing.T) {
		artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{Status: models.SuccessBackupStatus.Pointer()})
		require.NoError(t, err)

		ri, err := models.CreateRestoreHistoryItem(db.Querier, models.CreateRestoreHistoryItemParams{
			ArtifactID: artifact.ID,
			ServiceID:  *agent.ServiceID,
			Status:     models.InProgressRestoreStatus,
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			err := models.RemoveRestoreHistoryItem(db.Querier, ri.ID)
			require.NoError(t, err)
		})

		res, oldStatus, err := removalService.lockArtifact(artifact.ID, models.DeletingBackupStatus)
		assert.Nil(t, res)
		assert.Empty(t, oldStatus)
		assert.Contains(t, err.Error(), "artifact is used by currently running restore operation")

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.SuccessBackupStatus, artifact.Status)
	})

	t.Run("success", func(t *testing.T) {
		res, oldStatus, err := removalService.lockArtifact(artifact.ID, models.DeletingBackupStatus)
		require.NotNil(t, res)
		assert.Equal(t, models.SuccessBackupStatus, oldStatus)
		require.NoError(t, err)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.DeletingBackupStatus, artifact.Status)
	})
}

func TestReleaseArtifact(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	agent, _ := setup(t, db.Querier, models.MongoDBServiceType, "test-service3")

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			FilesystemConfig: &models.FilesystemLocationConfig{Path: "/"},
		},
	})
	require.NoError(t, err)

	artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:       "artifact_name",
		Vendor:     "MongoDB",
		LocationID: locationRes.ID,
		ServiceID:  *agent.ServiceID,
		DataModel:  models.LogicalDataModel,
		Mode:       models.PITR,
		Status:     models.DeletingBackupStatus,
		Folder:     "artifact_folder",
	})
	require.NoError(t, err)

	removalService := NewRemovalService(db, nil)

	t.Run("wrong releasing status", func(t *testing.T) {
		err := removalService.releaseArtifact(artifact.ID, models.PendingBackupStatus)
		assert.ErrorIs(t, err, ErrIncorrectArtifactStatus)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.DeletingBackupStatus, artifact.Status)
	})

	t.Run("success", func(t *testing.T) {
		err := removalService.releaseArtifact(artifact.ID, models.SuccessBackupStatus)
		assert.NoError(t, err)

		artifact, err = models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, models.SuccessBackupStatus, artifact.Status)
	})
}
