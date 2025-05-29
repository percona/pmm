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
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestEnsureRetention(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedRemovalService := &mockRemovalService{}
	retentionService := NewRetentionService(db, mockedRemovalService)

	agent, _ := setup(t, db.Querier, models.MySQLServiceType, "test-service")
	endpoint := "https://s3.us-west-2.amazonaws.com/"
	accessKey, secretKey, bucketName, bucketRegion := "access_key", "secret_key", "example_bucket", "us-east-2"

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: &models.S3LocationConfig{
				Endpoint:     endpoint,
				AccessKey:    accessKey,
				SecretKey:    secretKey,
				BucketName:   bucketName,
				BucketRegion: bucketRegion,
			},
		},
	})
	require.NoError(t, err)

	t.Run("wrong task mode", func(t *testing.T) {
		wrongModetask, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
			CronExpression: "* * * * *",
			Type:           models.ScheduledMongoDBBackupTask,
			Data: &models.ScheduledTaskData{
				MongoDBBackupTask: &models.MongoBackupTaskData{
					CommonBackupTaskData: models.CommonBackupTaskData{
						Name: "test",
						Mode: "wrong backup mode",
					},
				},
			},
		})
		require.NoError(t, err)

		// Returns nil, no dependency calls.
		err = retentionService.EnforceRetention(wrongModetask.ID)
		assert.NoError(t, err)
	})

	t.Run("successful snapshot", func(t *testing.T) {
		task, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
			CronExpression: "* * * * *",
			Type:           models.ScheduledMongoDBBackupTask,
			Data: &models.ScheduledTaskData{
				MongoDBBackupTask: &models.MongoBackupTaskData{
					CommonBackupTaskData: models.CommonBackupTaskData{
						ServiceID:  *agent.ServiceID,
						LocationID: locationRes.ID,
						Name:       "test2",
						Retention:  0,
						Mode:       models.Snapshot,
					},
				},
			},
			Disabled: false,
		})
		require.NoError(t, err)

		createArtifact := func() {
			_, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:       gofakeit.Name(),
				Vendor:     "MongoDB",
				LocationID: locationRes.ID,
				ServiceID:  *agent.ServiceID,
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Snapshot,
				Status:     models.SuccessBackupStatus,
				ScheduleID: task.ID,
			})
			require.NoError(t, err)
		}
		require.NoError(t, err)

		countArtifacts := func() int {
			artifacts, err := models.FindArtifacts(db.Querier, models.ArtifactFilters{
				ScheduleID: task.ID,
				Status:     models.SuccessBackupStatus,
			})
			require.NoError(t, err)
			return len(artifacts)
		}

		deleteArtifacts := func(_ mock.Arguments) {
			artifacts, err := models.FindArtifacts(db.Querier, models.ArtifactFilters{
				ScheduleID: task.ID,
				Status:     models.SuccessBackupStatus,
			})
			require.NoError(t, err)
			require.NotEmpty(t, artifacts)

			err = models.DeleteArtifact(db.Querier, artifacts[0].ID)
			require.NoError(t, err)
		}

		changeRetention := func(retention uint32) {
			task.Data.MongoDBBackupTask.Retention = retention
			task, err = models.ChangeScheduledTask(db.Querier, task.ID, models.ChangeScheduledTaskParams{
				Data: task.Data,
			})
			require.NoError(t, err)
		}

		createArtifact()
		assert.Equal(t, 1, countArtifacts())
		createArtifact()
		assert.NoError(t, retentionService.EnforceRetention(task.ID))
		assert.Equal(t, 2, countArtifacts())

		createArtifact()
		createArtifact()
		createArtifact()
		assert.NoError(t, retentionService.EnforceRetention(task.ID))
		assert.Equal(t, 5, countArtifacts())

		changeRetention(6)
		assert.NoError(t, retentionService.EnforceRetention(task.ID))
		assert.Equal(t, 5, countArtifacts())

		changeRetention(4)
		mockedRemovalService.On("DeleteArtifact", mock.Anything, mock.Anything, true).Return(nil).Run(deleteArtifacts).Once()
		assert.NoError(t, retentionService.EnforceRetention(task.ID))
		assert.Equal(t, 4, countArtifacts())

		changeRetention(2)
		mockedRemovalService.On("DeleteArtifact", mock.Anything, mock.Anything, true).Return(nil).Run(deleteArtifacts).Twice()
		assert.NoError(t, retentionService.EnforceRetention(task.ID))
		assert.Equal(t, 2, countArtifacts())
	})

	t.Run("pitr", func(t *testing.T) {
		task, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
			CronExpression: "* * * * *",
			Type:           models.ScheduledMongoDBBackupTask,
			Data: &models.ScheduledTaskData{
				MongoDBBackupTask: &models.MongoBackupTaskData{
					CommonBackupTaskData: models.CommonBackupTaskData{
						ServiceID:  *agent.ServiceID,
						LocationID: locationRes.ID,
						Name:       "test3",
						Retention:  5,
						Mode:       models.PITR,
					},
				},
			},
			Disabled: false,
		})
		require.NoError(t, err)

		t.Run("successful", func(t *testing.T) {
			artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:       gofakeit.Name(),
				Vendor:     "MongoDB",
				LocationID: locationRes.ID,
				ServiceID:  *agent.ServiceID,
				DataModel:  models.LogicalDataModel,
				Mode:       models.PITR,
				Status:     models.SuccessBackupStatus,
				ScheduleID: task.ID,
			})
			require.NoError(t, err)

			for i := 1; i <= 5; i++ {
				_, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{Metadata: &models.Metadata{FileList: []models.File{{Name: "file"}}}})
				require.NoError(t, err)
			}

			// Retention equals Metadata list length, no dependency call.
			err = retentionService.EnforceRetention(task.ID)
			require.NoError(t, err)

			taskData := task.Data
			taskData.MongoDBBackupTask.Retention = 3
			_, err = models.ChangeScheduledTask(db.Querier, task.ID, models.ChangeScheduledTaskParams{Data: taskData})
			require.NoError(t, err)

			// Must trim 2 elements from Metadata.
			mockedRemovalService.On("TrimPITRArtifact", mock.Anything, artifact.ID, 2).Return(nil).Once()

			err = retentionService.EnforceRetention(task.ID)
			require.NoError(t, err)
		})

		t.Run("more than one pitr artifact", func(t *testing.T) {
			_, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:       gofakeit.Name(),
				Vendor:     "MongoDB",
				LocationID: locationRes.ID,
				ServiceID:  *agent.ServiceID,
				DataModel:  models.LogicalDataModel,
				Mode:       models.PITR,
				Status:     models.SuccessBackupStatus,
				ScheduleID: task.ID,
			})
			require.NoError(t, err)

			err = retentionService.EnforceRetention(task.ID)
			require.Error(t, err)
			assert.Equal(t, "Can be only one artifact entity for PITR in the database but found 2", err.Error())
		})
	})

	mockedRemovalService.AssertExpectations(t)
}
