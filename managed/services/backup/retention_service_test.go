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
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	mockedS3 := &mockS3{}
	removalService := NewRemovalService(db, mockedS3)
	retentionService := NewRetentionService(db, removalService)
	mockedS3.On("RemoveRecursive", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(nil)

	agent := setup(t, db.Querier, models.MySQLServiceType, "test-service")
	endpoint := "https://s3.us-west-2.amazonaws.com/"
	accessKey, secretKey, bucketName, bucketRegion := "access_key", "secret_key", "example_bucket", "us-east-2"

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

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

	task, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMongoDBBackupTask,
		Data: &models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:  *agent.ServiceID,
					LocationID: locationRes.ID,
					Name:       "test",
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
	assert.NoError(t, retentionService.EnforceRetention(ctx, task.ID))
	assert.Equal(t, 2, countArtifacts())

	createArtifact()
	createArtifact()
	createArtifact()
	assert.NoError(t, retentionService.EnforceRetention(ctx, task.ID))
	assert.Equal(t, 5, countArtifacts())

	changeRetention(6)
	assert.NoError(t, retentionService.EnforceRetention(ctx, task.ID))
	assert.Equal(t, 5, countArtifacts())

	changeRetention(4)
	assert.NoError(t, retentionService.EnforceRetention(ctx, task.ID))
	assert.Equal(t, 4, countArtifacts())

	changeRetention(2)
	assert.NoError(t, retentionService.EnforceRetention(ctx, task.ID))
	assert.Equal(t, 2, countArtifacts())
}
