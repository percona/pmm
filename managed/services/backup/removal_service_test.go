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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestDeleteArtifact(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	removalService := NewRemovalService(db)

	agent := setup(t, db.Querier, models.MySQLServiceType, "test-service")
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

	artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
		Name:       "artifact_name",
		Vendor:     "MySQL",
		LocationID: locationRes.ID,
		ServiceID:  *agent.ServiceID,
		DataModel:  models.PhysicalDataModel,
		Mode:       models.Snapshot,
		Status:     models.PendingBackupStatus,
	})
	require.NoError(t, err)

	t.Run("artifact not in final status", func(t *testing.T) {
		err := removalService.DeleteArtifact(ctx, artifact.ID, false)
		require.Contains(t, err.Error(), "isn't in the final state")

		artifact, err := models.FindArtifactByID(db.Querier, artifact.ID)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		assert.Equal(t, artifact.Status, models.PendingBackupStatus)
	})

	t.Run("successful delete", func(t *testing.T) {
		artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{Status: models.BackupStatusPointer(models.SuccessBackupStatus)})
		err = removalService.DeleteArtifact(ctx, artifact.ID, false)
		assert.NoError(t, err)

		_, err := models.FindArtifactByID(db.Querier, artifact.ID)
		assert.ErrorIs(t, err, models.ErrNotFound)
	})
}
