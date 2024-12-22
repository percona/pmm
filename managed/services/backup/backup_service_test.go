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
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setup(t *testing.T, q *reform.Querier, serviceType models.ServiceType, serviceName string) (*models.Agent, *models.Service) {
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
	return agent, service
}

func TestPerformBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedAgentService := &mockAgentService{}
	mockedCompatibilityService := &mockCompatibilityService{}
	backupService := NewService(db, mockedJobsService, mockedAgentService, mockedCompatibilityService, nil)

	s3Location, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test s3 location",
		Description: "Test s3 description",
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

	filesystemLocation, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test local location",
		Description: "Test local description",
		BackupLocationConfig: models.BackupLocationConfig{
			FilesystemConfig: &models.FilesystemLocationConfig{
				Path: "/opt/data",
			},
		},
	})
	require.NoError(t, err)

	t.Run("mysql", func(t *testing.T) {
		agent, _ := setup(t, db.Querier, models.MySQLServiceType, "test-mysql-backup-service")

		for _, tc := range []struct {
			name                                        string
			dbVersion                                   string
			locationModel                               *models.BackupLocation
			dataModel                                   models.DataModel
			errFromCheckSoftwareCompatibilityForService bool
			expectedError                               error
		}{
			{
				name:          "successful",
				dbVersion:     "8.0.25",
				locationModel: s3Location,
				dataModel:     models.PhysicalDataModel,
				expectedError: nil,
			},
			{
				name:          "fail",
				dbVersion:     "",
				locationModel: s3Location,
				dataModel:     models.PhysicalDataModel,
				errFromCheckSoftwareCompatibilityForService: true,
				expectedError: ErrXtrabackupNotInstalled,
			},
			{
				name:          "unsupported data model",
				dbVersion:     "8.0.25",
				locationModel: s3Location,
				dataModel:     models.LogicalDataModel,
				expectedError: ErrIncompatibleDataModel,
			},
			{
				name:          "unsupported location type",
				dbVersion:     "8.0.25",
				locationModel: filesystemLocation,
				dataModel:     models.PhysicalDataModel,
				expectedError: ErrIncompatibleLocationType,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var retErr error
				if tc.errFromCheckSoftwareCompatibilityForService {
					retErr = tc.expectedError
				}
				mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
					Return(tc.dbVersion, retErr).Once()

				if tc.expectedError == nil {
					locationConfig := &models.BackupLocationConfig{
						FilesystemConfig: tc.locationModel.FilesystemConfig,
						S3Config:         tc.locationModel.S3Config,
					}
					mockedJobsService.On("StartMySQLBackupJob", mock.Anything, pointer.GetString(agent.PMMAgentID), time.Duration(0),
						mock.Anything, mock.Anything, locationConfig, "artifact_folder").Return(nil).Once()
				}

				artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
					ServiceID:  pointer.GetString(agent.ServiceID),
					LocationID: tc.locationModel.ID,
					Name:       tc.name + "_" + "test_backup",
					DataModel:  tc.dataModel,
					Mode:       models.Snapshot,
					Folder:     "artifact_folder",
				})

				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Empty(t, artifactID)
					return
				}

				assert.NoError(t, err)
				artifact, err := models.FindArtifactByID(db.Querier, artifactID)
				require.NoError(t, err)
				assert.Equal(t, tc.locationModel.ID, artifact.LocationID)
				assert.Equal(t, *agent.ServiceID, artifact.ServiceID)
				assert.EqualValues(t, models.MySQLServiceType, artifact.Vendor)
			})
		}
	})

	t.Run("mongodb", func(t *testing.T) {
		agent, _ := setup(t, db.Querier, models.MongoDBServiceType, "test-mongo-backup-service")

		t.Run("PITR is incompatible with physical backups", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("", nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  pointer.GetString(agent.ServiceID),
				LocationID: s3Location.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.PITR,
				Folder:     "artifact_folder_2",
			})
			assert.ErrorIs(t, err, ErrIncompatibleDataModel)
			assert.Empty(t, artifactID)
		})

		t.Run("backup fails for empty service ID", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, "").Return("", nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  "",
				LocationID: s3Location.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.PITR,
				Folder:     "artifact_folder_3",
			})
			assert.ErrorContains(t, err, "Empty Service ID")
			assert.Empty(t, artifactID)
		})

		t.Run("Incremental backups fails for MongoDB", func(t *testing.T) {
			mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
				Return("", nil).Once()
			artifactID, err := backupService.PerformBackup(ctx, PerformBackupParams{
				ServiceID:  pointer.GetString(agent.ServiceID),
				LocationID: s3Location.ID,
				Name:       "test_backup",
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Incremental,
				Folder:     "artifact_folder_4",
			})
			assert.ErrorContains(t, err, "the only supported backups mode for mongoDB is snapshot and PITR")
			assert.Empty(t, artifactID)
		})
	})

	mock.AssertExpectationsForObjects(t, mockedJobsService, mockedCompatibilityService)
}

func TestRestoreBackup(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedJobsService := &mockJobsService{}
	mockedAgentService := &mockAgentService{}
	mockedCompatibilityService := &mockCompatibilityService{}
	backupService := NewService(db, mockedJobsService, mockedAgentService, mockedCompatibilityService, nil)

	artifactFolder := "artifact_folder"

	s3Location, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
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

	filesystemLocation, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
		Name:        "Test local location",
		Description: "Test local description",
		BackupLocationConfig: models.BackupLocationConfig{
			FilesystemConfig: &models.FilesystemLocationConfig{
				Path: "/opt/data",
			},
		},
	})
	require.NoError(t, err)

	t.Run("mysql", func(t *testing.T) {
		agent, _ := setup(t, db.Querier, models.MySQLServiceType, "test-mysql-restore-service")
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "mysql-artifact-name",
			Vendor:     string(models.MySQLServiceType),
			DBVersion:  "8.0.25",
			LocationID: s3Location.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.PhysicalDataModel,
			Mode:       models.Snapshot,
			Status:     models.SuccessBackupStatus,
			Folder:     artifactFolder,
		})
		require.NoError(t, err)

		artifact, err = models.UpdateArtifact(db.Querier, artifact.ID, models.UpdateArtifactParams{
			Metadata: &models.Metadata{FileList: []models.File{{Name: "test_file_name"}}},
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
				name:          "incompatible version",
				dbVersion:     "",
				expectedError: ErrIncompatibleTargetMySQL,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
					Return(tc.dbVersion, nil).Once()
				mockedCompatibilityService.On("CheckArtifactCompatibility", artifact.ID, tc.dbVersion).Return(tc.expectedError).Once()

				if tc.expectedError == nil {
					mockedJobsService.On("StartMySQLRestoreBackupJob", mock.Anything, pointer.GetString(agent.PMMAgentID),
						pointer.GetString(agent.ServiceID), mock.Anything, artifact.Name, mock.Anything, artifactFolder).Return(nil).Once()
				}
				restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID, time.Unix(0, 0))
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
				Status: models.PendingBackupStatus.Pointer(),
			})
			require.NoError(t, err)
			require.NotNil(t, updatedArtifact)

			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID, time.Unix(0, 0))
			require.ErrorIs(t, err, ErrArtifactNotReady)
			assert.Empty(t, restoreID)
		})
	})

	t.Run("mongo", func(t *testing.T) {
		agent, service := setup(t, db.Querier, models.MongoDBServiceType, "test-mongo-restore-service")
		artifactWithVersion, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "mongodb-artifact-name-version",
			Vendor:     string(models.MongoDBSoftwareName),
			DBVersion:  "6.0.2-1",
			LocationID: s3Location.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.LogicalDataModel,
			Mode:       models.Snapshot,
			Status:     models.SuccessBackupStatus,
			Folder:     artifactFolder,
		})
		require.NoError(t, err)

		artifactWithVersion, err = models.UpdateArtifact(db.Querier, artifactWithVersion.ID, models.UpdateArtifactParams{
			Metadata: &models.Metadata{BackupToolData: &models.BackupToolData{PbmMetadata: &models.PbmMetadata{Name: "artifact_repr_name"}}},
		})
		require.NoError(t, err)

		artifactNoVersion, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "mongodb-artifact-name-no-version",
			Vendor:     string(models.MongoDBSoftwareName),
			LocationID: s3Location.ID,
			ServiceID:  *agent.ServiceID,
			DataModel:  models.LogicalDataModel,
			Mode:       models.Snapshot,
			Status:     models.SuccessBackupStatus,
		})
		require.NoError(t, err)

		for _, tc := range []struct {
			name          string
			artifact      *models.Artifact
			dbVersion     string
			expectedError error
		}{
			{
				name:          "successful",
				artifact:      artifactWithVersion,
				dbVersion:     "6.0.2-1",
				expectedError: nil,
			},
			{
				name:          "incompatible version",
				artifact:      artifactWithVersion,
				dbVersion:     "6.0.2-3",
				expectedError: ErrIncompatibleTargetMongoDB,
			},
			{
				name:          "empty db version",
				artifact:      artifactWithVersion,
				dbVersion:     "",
				expectedError: ErrIncompatibleTargetMongoDB,
			},
			{
				name:          "success if artifact has no version",
				artifact:      artifactNoVersion,
				dbVersion:     "6.0.2-1",
				expectedError: nil,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mockedCompatibilityService.On("CheckSoftwareCompatibilityForService", ctx, pointer.GetString(agent.ServiceID)).
					Return(tc.dbVersion, nil).Once()
				mockedCompatibilityService.On("CheckArtifactCompatibility", tc.artifact.ID, tc.dbVersion).Return(tc.expectedError).Once()

				if tc.expectedError == nil {
					if len(tc.artifact.MetadataList) != 0 && tc.artifact.MetadataList[0].BackupToolData != nil {
						mockedJobsService.On("StartMongoDBRestoreBackupJob", service, mock.Anything, pointer.GetString(agent.PMMAgentID),
							time.Duration(0), tc.artifact.Name, tc.artifact.MetadataList[0].BackupToolData.PbmMetadata.Name, tc.artifact.DataModel,
							mock.Anything, time.Unix(0, 0), tc.artifact.Folder).Return(nil).Once()
					} else {
						mockedJobsService.On("StartMongoDBRestoreBackupJob", service, mock.Anything, pointer.GetString(agent.PMMAgentID),
							time.Duration(0), tc.artifact.Name, "", tc.artifact.DataModel,
							mock.Anything, time.Unix(0, 0), tc.artifact.Folder).Return(nil).Once()
					}
				}
				restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), tc.artifact.ID, time.Unix(0, 0))
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
			artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:       "mongo-artifact-name-s3",
				Vendor:     string(models.MongoDBServiceType),
				LocationID: s3Location.ID,
				ServiceID:  *agent.ServiceID,
				DataModel:  models.LogicalDataModel,
				Mode:       models.Snapshot,
				Status:     models.PendingBackupStatus,
			})
			require.NoError(t, err)

			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID, time.Unix(0, 0))
			require.ErrorIs(t, err, ErrArtifactNotReady)
			assert.Empty(t, restoreID)
		})

		t.Run("PITR not supported for local storages", func(t *testing.T) {
			artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:       "mongo-artifact-name-local",
				Vendor:     string(models.MongoDBServiceType),
				LocationID: filesystemLocation.ID,
				ServiceID:  *agent.ServiceID,
				DataModel:  models.LogicalDataModel,
				Mode:       models.PITR,
				Status:     models.SuccessBackupStatus,
			})
			require.NoError(t, err)

			restoreID, err := backupService.RestoreBackup(ctx, pointer.GetString(agent.ServiceID), artifact.ID, time.Now())
			require.ErrorIs(t, err, ErrIncompatibleLocationType)
			assert.Empty(t, restoreID)
		})
	})

	mock.AssertExpectationsForObjects(t, mockedJobsService, mockedAgentService, mockedCompatibilityService)
}

func TestCheckArtifactModePreconditions(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)

	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	mockedPbmPITRService := &mockPbmPITRService{}
	backupService := NewService(db, nil, nil, nil, mockedPbmPITRService)

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
		agent, _ := setup(t, db.Querier, models.MySQLServiceType, "test-mysql-restore-service")

		for _, tc := range []struct {
			name           string
			pitrValue      time.Time
			artifactParams models.CreateArtifactParams
			err            error
		}{
			{
				name:      "success",
				pitrValue: time.Unix(0, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mysql-artifact-name-1",
					Vendor:     string(models.MySQLServiceType),
					DBVersion:  "8.0.25",
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
					Status:     models.SuccessBackupStatus,
				},
				err: nil,
			},
			{
				name:      "PITR not supported for MySQL",
				pitrValue: time.Unix(0, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mysql-artifact-name-2",
					Vendor:     string(models.MySQLServiceType),
					DBVersion:  "8.0.25",
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.PhysicalDataModel,
					Mode:       models.PITR,
					Status:     models.SuccessBackupStatus,
				},
				err: ErrIncompatibleService,
			},
			{
				name:      "snapshot artifact is not compatible with non-empty pitr date",
				pitrValue: time.Unix(1, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mysql-artifact-name-3",
					Vendor:     string(models.MySQLServiceType),
					DBVersion:  "8.0.25",
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
					Status:     models.SuccessBackupStatus,
				},
				err: ErrIncompatibleArtifactMode,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				artifact, err := models.CreateArtifact(db.Querier, tc.artifactParams)
				require.NoError(t, err)

				err = backupService.checkArtifactModePreconditions(ctx, artifact.ID, tc.pitrValue)
				if tc.err == nil {
					require.NoError(t, err)
				} else {
					assert.ErrorIs(t, err, tc.err)
				}
			})
		}
	})

	t.Run("mongo", func(t *testing.T) {
		agent, _ := setup(t, db.Querier, models.MongoDBServiceType, "test-mongodb-restore-service")

		rangeStart1 := uint32(1)
		rangeEnd1 := rangeStart1 + (60 * 60 * 3) // plus 3 hours

		rangeStart2 := uint32(time.Now().Unix())
		rangeEnd2 := rangeStart2 + (60 * 60 * 3) // plus 3 hours

		timelineList := []Timeline{
			{Start: rangeStart1, End: rangeEnd1},
			{Start: rangeStart2, End: rangeEnd2},
		}

		for _, tc := range []struct {
			name           string
			pitrValue      time.Time
			prepareMock    bool
			artifactParams models.CreateArtifactParams
			err            error
		}{
			{
				name:      "success logical restore",
				pitrValue: time.Unix(0, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mongo-artifact-name-1",
					Vendor:     string(models.MongoDBServiceType),
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.LogicalDataModel,
					Mode:       models.Snapshot,
					Status:     models.SuccessBackupStatus,
				},
				err: nil,
			},
			{
				name:      "physical restore is supported",
				pitrValue: time.Unix(0, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mongo-artifact-name-2",
					Vendor:     string(models.MongoDBServiceType),
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
					Status:     models.SuccessBackupStatus,
				},
				err: nil,
			},
			{
				name:      "snapshot artifact is not compatible with non-empty pitr date",
				pitrValue: time.Unix(1, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mongo-artifact-name-3",
					Vendor:     string(models.MongoDBServiceType),
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.LogicalDataModel,
					Mode:       models.Snapshot,
					Status:     models.SuccessBackupStatus,
				},
				err: ErrIncompatibleArtifactMode,
			},
			{
				name:      "timestamp not provided for pitr artifact",
				pitrValue: time.Unix(0, 0),
				artifactParams: models.CreateArtifactParams{
					Name:       "mongo-artifact-name-4",
					Vendor:     string(models.MongoDBServiceType),
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.LogicalDataModel,
					Mode:       models.PITR,
					Status:     models.SuccessBackupStatus,
				},
				err: ErrIncompatibleArtifactMode,
			},
			{
				name:        "pitr timestamp out of range",
				pitrValue:   time.Unix(int64(rangeStart2)-1, 0),
				prepareMock: true,
				artifactParams: models.CreateArtifactParams{
					Name:       "mongo-artifact-name-5",
					Vendor:     string(models.MongoDBServiceType),
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.LogicalDataModel,
					Mode:       models.PITR,
					Status:     models.SuccessBackupStatus,
				},
				err: ErrTimestampOutOfRange,
			},
			{
				name:        "success pitr timestamp inside the range",
				pitrValue:   time.Unix(int64(rangeStart2)+1, 0),
				prepareMock: true,
				artifactParams: models.CreateArtifactParams{
					Name:       "mongo-artifact-name-6",
					Vendor:     string(models.MongoDBServiceType),
					LocationID: locationRes.ID,
					ServiceID:  *agent.ServiceID,
					DataModel:  models.LogicalDataModel,
					Mode:       models.PITR,
					Status:     models.SuccessBackupStatus,
				},
				err: nil,
			},
			{
				name:      "sharded cluster restore not supported",
				pitrValue: time.Unix(0, 0),
				artifactParams: models.CreateArtifactParams{
					Name:             "mongo-artifact-name-7",
					Vendor:           string(models.MongoDBServiceType),
					LocationID:       locationRes.ID,
					ServiceID:        *agent.ServiceID,
					DataModel:        models.LogicalDataModel,
					Mode:             models.Snapshot,
					Status:           models.SuccessBackupStatus,
					IsShardedCluster: true,
				},
				err: ErrIncompatibleService,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				artifact, err := models.CreateArtifact(db.Querier, tc.artifactParams)
				require.NoError(t, err)

				if tc.prepareMock {
					mockedPbmPITRService.On("ListPITRTimeranges", ctx, mock.Anything, locationRes, artifact).Return(timelineList, nil).Once()
				}

				err = backupService.checkArtifactModePreconditions(ctx, artifact.ID, tc.pitrValue)
				if tc.err == nil {
					require.NoError(t, err)
				} else {
					assert.ErrorIs(t, err, tc.err)
				}
			})
		}
	})

	mock.AssertExpectationsForObjects(t, mockedPbmPITRService)
}

func TestInTimeSpan(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name    string
		start   time.Time
		end     time.Time
		value   time.Time
		inRange bool
	}{
		{
			name:    "success start lt end",
			start:   now.Add(-1 * time.Hour),
			end:     now.Add(1 * time.Hour),
			value:   now,
			inRange: true,
		},
		{
			name:    "success start eq end",
			start:   now,
			end:     now,
			value:   now,
			inRange: true,
		},
		{
			name:    "fail start gt end",
			start:   now.Add(1 * time.Hour),
			end:     now.Add(-1 * time.Hour),
			value:   now,
			inRange: false,
		},
		{
			name:    "out of range",
			start:   now.Add(-1 * time.Hour),
			end:     now.Add(1 * time.Hour),
			value:   now.Add(1 * time.Hour).Add(1 * time.Second),
			inRange: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := inTimeSpan(tc.start, tc.end, tc.value)
			assert.Equal(t, tc.inRange, res)
		})
	}
}
