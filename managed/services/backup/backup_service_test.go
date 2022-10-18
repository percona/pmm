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

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setup(t *testing.T, q *reform.Querier, serviceType models.ServiceType, serviceName string) *models.Agent {
	t.Helper()
	require.Contains(t, []models.ServiceType{models.MySQLServiceType, models.MongoDBServiceType}, serviceType)

	node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node-" + t.Name(),
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(q, node.NodeID, nil)
	require.NoError(t, err)

	var service *models.Service
	service, err = models.AddNewService(q, serviceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(60000),
	})
	require.NoError(t, err)

	agentType := models.MySQLdExporterType
	if serviceType == models.MongoDBServiceType {
		agentType = models.MongoDBExporterType
	}

	agent, err := models.CreateAgent(q, agentType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  service.ServiceID,
		Username:   "user",
		Password:   "password",
	})
	require.NoError(t, err)
	return agent
}

func TestPerformBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedAgentService := &mockAgentService{}
	mockedCompatibilityService := &mockCompatibilityService{}
	backupService := NewService(db, mockedJobsService, mockedAgentService, mockedCompatibilityService)

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: &models.S3LocationConfig{
				Endpoint:     "https://s3.us-west-2.amazonaws.com/",
				AccessKey:    "access_key",
				SecretKey:    "secret_key",
				BucketName:   "example_bucket",
				BucketRegion: "us-east-2",
			},
		},
	})
	require.NoError(t, err)

	t.Run("mysql", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MySQLServiceType, "test-mysql-backup-service")
		mockedJobsService.On("StartMySQLBackupJob", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(nil)

		for _, tc := range []struct {
			name          string
			dbVersion     string
			expectedError error
		}{
			{
				name:          "successful",
				dbVersion:     "8.0.25",
				expectedError: nil,
			},
			{
				name:          "fail",
				dbVersion:     "",
				expectedError: ErrXtrabackupNotInstalled,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
					Return(tc.dbVersion, tc.expectedError).Once()
				artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
					ServiceID:  pointer.GetString(agent.ServiceID),
					LocationID: locationRes.ID,
					Name:       "test_backup",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				})

				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Empty(t, artifactID)
					return
				}

				assert.NoError(t, err)
				artifact, err := models.FindArtifactByID(db.Querier, artifactID)
				require.NoError(t, err)
				assert.Equal(t, locationRes.ID, artifact.LocationID)
				assert.Equal(t, *agent.ServiceID, artifact.ServiceID)
				assert.EqualValues(t, models.MySQLServiceType, artifact.Vendor)
			})
		}
	})

	t.Run("mongodb", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MongoDBServiceType, "test-mongo-backup-service")

		t.Run("PITR is incompatible with physical backups", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("", nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  pointer.GetString(agent.ServiceID),
				LocationID: locationRes.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.PITR,
			})
			assert.ErrorIs(t, err, ErrIncompatibleDataModel)
			assert.Empty(t, artifactID)
		})

		t.Run("backup fails for empty service ID", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, "").Return("", nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  "",
				LocationID: locationRes.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.PITR,
			})
			assert.ErrorContains(t, err, "Empty Service ID")
			assert.Empty(t, artifactID)
		})

		t.Run("Incremental backups fails for MongoDB", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("", nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  pointer.GetString(agent.ServiceID),
				LocationID: locationRes.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Incremental,
			})
			assert.ErrorContains(t, err, "the only supported backups mode for mongoDB is snapshot and PITR")
			assert.Empty(t, artifactID)
		})
	})

	mock.AssertExpectationsForObjects(t, mockedJobsService, mockedAgentService, mockedCompatibilityService)
}

func TestRestoreBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedAgentService := &mockAgentService{}
	mockedCompatibilityService := &mockCompatibilityService{}
	backupService := NewService(db, mockedJobsService, mockedAgentService, mockedCompatibilityService)

	locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test location",
		Description: "Test description",
		BackupLocationConfig: models.BackupLocationConfig{
			S3Config: &models.S3LocationConfig{
				Endpoint:     "https://s3.us-west-2.amazonaws.com/",
				AccessKey:    "access_key",
				SecretKey:    "secret_key",
				BucketName:   "example_bucket",
				BucketRegion: "us-east-2",
			},
		},
	})
	require.NoError(t, err)

	t.Run("mysql", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MySQLServiceType, "test-mysql-restore-service")
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "mysql-artifact-name",
			Vendor:     string(models.MySQLServiceType),
			DBVersion:  "8.0.25",
			LocationID: locationRes.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.PhysicalDataModel,
			Mode:       models.Snapshot,
			Status:     models.SuccessBackupStatus,
		})
		require.NoError(t, err)

		for _, tc := range []struct {
			name          string
			dbVersion     string
			expectedError error
		}{
			{
				name:          "successful",
				dbVersion:     "8.0.25",
				expectedError: nil,
			},
			{
				name:          "fail",
				dbVersion:     "",
				expectedError: ErrXtrabackupNotInstalled,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
					Return(tc.dbVersion, tc.expectedError).Once()
				if tc.expectedError == nil {
					mockedJobsService.On("StartMySQLRestoreBackupJob", mock.Anything, pointer.GetString(agent.PMMAgentID),
						pointer.GetString(agent.ServiceID), mock.Anything, artifact.Name, mock.Anything).Return(nil).Once()
				}
				restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Empty(t, restoreID)
				} else {
					assert.NoError(t, err)
					assert.NotEmpty(t, restoreID)
				}
			})
		}

		t.Run("artifact not ready", func(t *testing.T) {
			updatedArtifact, err := models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.PendingBackupStatus),
			})
			require.NoError(t, err)
			require.NotNil(t, updatedArtifact)

			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("8.0.25", nil).Once()
			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
			require.Errorf(t, err, "artifact %q status is not successful, status: \"pending\"", artifact.ID)
			assert.Empty(t, restoreID)
		})
	})

	t.Run("mongo", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MongoDBServiceType, "test-mongo-restore-service")

		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "mongo-artifact-name",
			Vendor:     string(models.MongoDBServiceType),
			LocationID: locationRes.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.PhysicalDataModel,
			Mode:       models.Snapshot,
			Status:     models.PendingBackupStatus,
		})
		require.NoError(t, err)

		t.Run("incomplete backups won't restore", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("", nil).Once()

			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
			require.Errorf(t, err, "artifact %q status is not successful, status: \"pending\"", artifact.ID)
			assert.Empty(t, restoreID)
		})

		t.Run("physical backups is not supported", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("", nil).Once()

			_, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.SuccessBackupStatus),
			})
			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID)
			require.ErrorIs(t, err, ErrIncompatibleService)
			assert.Empty(t, restoreID)
		})
	})

	mock.AssertExpectationsForObjects(t, mockedJobsService, mockedAgentService, mockedCompatibilityService)
}
