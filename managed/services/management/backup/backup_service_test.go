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

	"github.com/AlekSi/pointer"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	backupv1 "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/scheduler"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func setup(t *testing.T, q *reform.Querier, serviceType models.ServiceType, serviceName, clusterName string) *models.Agent { //nolint:unparam
	t.Helper()
	require.Contains(t, []models.ServiceType{models.MySQLServiceType, models.MongoDBServiceType}, serviceType)
	node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node-" + t.Name(),
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(q, node.NodeID, nil)
	require.NoError(t, err)
	require.NoError(t, q.Update(pmmAgent))

	var service *models.Service
	service, err = models.AddNewService(q, serviceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		Cluster:     clusterName,
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

func TestStartBackup(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		backupService := &mockBackupService{}
		mockedPbmPITRService := &mockPbmPITRService{}
		sqlDB := testdb.Open(t, database.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		backupSvc := NewBackupsService(db, backupService, nil, nil, nil, mockedPbmPITRService)
		agent := setup(t, db.Querier, models.MySQLServiceType, t.Name(), "cluster")

		for _, tc := range []struct {
			testName    string
			backupError error
			code        backupv1.ErrorCode
		}{
			{
				testName:    "xtrabackup not installed",
				backupError: backup.ErrXtrabackupNotInstalled,
				code:        backupv1.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED,
			},
			{
				testName:    "invalid xtrabackup",
				backupError: backup.ErrInvalidXtrabackup,
				code:        backupv1.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP,
			},
			{
				testName:    "incompatible xtrabackup",
				backupError: backup.ErrIncompatibleXtrabackup,
				code:        backupv1.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP,
			},
		} {
			t.Run(tc.testName, func(t *testing.T) {
				backupError := fmt.Errorf("error: %w", tc.backupError)
				backupService.On("PerformBackup", mock.Anything, mock.Anything).
					Return("", backupError).Once()
				ctx := context.Background()
				resp, err := backupSvc.StartBackup(ctx, &backupv1.StartBackupRequest{
					ServiceId:     *agent.ServiceID,
					LocationId:    "locationID",
					Name:          "name",
					Description:   "description",
					DataModel:     backupv1.DataModel_DATA_MODEL_PHYSICAL,
					RetryInterval: nil,
					Retries:       0,
				})
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.FailedPrecondition, st.Code())
				assert.Equal(t, backupError.Error(), st.Message())
				require.Len(t, st.Details(), 1)
				detailedError, ok := st.Details()[0].(*backupv1.Error)
				require.True(t, ok)
				assert.Equal(t, tc.code, detailedError.Code)
			})
		}
	})

	t.Run("mongodb", func(t *testing.T) {
		sqlDB := testdb.Open(t, database.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		agent := setup(t, db.Querier, models.MongoDBServiceType, t.Name(), "cluster")

		locationRes, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
			Name:        "Test location snapshots",
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

		t.Run("starting mongodb physical snapshot is successful", func(t *testing.T) {
			ctx := context.Background()
			backupService := &mockBackupService{}
			mockedPbmPITRService := &mockPbmPITRService{}
			backupSvc := NewBackupsService(db, backupService, nil, nil, nil, mockedPbmPITRService)
			backupService.On("PerformBackup", mock.Anything, mock.Anything).Return("", nil)
			_, err := backupSvc.StartBackup(ctx, &backupv1.StartBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    locationRes.ID,
				Name:          "name",
				Description:   "description",
				RetryInterval: nil,
				Retries:       0,
				DataModel:     backupv1.DataModel_DATA_MODEL_PHYSICAL,
			})
			require.NoError(t, err)
		})

		t.Run("check folder and artifact name", func(t *testing.T) {
			ctx := context.Background()
			backupService := &mockBackupService{}
			mockedPbmPITRService := &mockPbmPITRService{}

			backupSvc := NewBackupsService(db, backupService, nil, nil, nil, mockedPbmPITRService)

			tc := []struct {
				TestName   string
				BackupName string
				Folder     string
				ErrString  string
			}{
				{
					TestName:   "normal",
					BackupName: ".normal_name:1-",
					Folder:     ".normal_folder:1-/tmp",
					ErrString:  "",
				},
				{
					TestName:   "not allowed symbols in name",
					BackupName: "normal/name",
					Folder:     "normal_folder",
					ErrString:  "rpc error: code = InvalidArgument desc = Backup name can contain only dots, colons, letters, digits, underscores and dashes.",
				},
				{
					TestName:   "not allowed symbols in folder",
					BackupName: "normal_name",
					Folder:     "$._folder:1-/tmp",
					ErrString:  "rpc error: code = InvalidArgument desc = Folder name can contain only dots, colons, slashes, letters, digits, underscores and dashes.",
				},
				{
					TestName:   "folder refers to a parent directory",
					BackupName: "normal_name",
					Folder:     "../../../some_folder",
					ErrString:  "rpc error: code = InvalidArgument desc = Specified folder refers to a parent directory.",
				},
				{
					TestName:   "folder points to absolute path",
					BackupName: "normal_name",
					Folder:     "/some_folder",
					ErrString:  "rpc error: code = InvalidArgument desc = Folder should be a relative path (shouldn't contain leading slashes).",
				},
				{
					TestName:   "folder in non-canonical format",
					BackupName: "normal_name",
					Folder:     "some_folder/../../../../root",
					ErrString:  "rpc error: code = InvalidArgument desc = Specified folder in non-canonical format, canonical would be: \"../../../root\".",
				},
			}

			for _, test := range tc {
				t.Run(test.TestName, func(t *testing.T) {
					if test.ErrString == "" {
						backupService.On("PerformBackup", mock.Anything, mock.Anything).Return("", nil).Once()
					}
					res, err := backupSvc.StartBackup(ctx, &backupv1.StartBackupRequest{
						Name:      test.BackupName,
						Folder:    test.Folder,
						ServiceId: *agent.ServiceID,
						DataModel: backupv1.DataModel_DATA_MODEL_LOGICAL,
					})
					if test.ErrString != "" {
						assert.Nil(t, res)
						assert.Equal(t, test.ErrString, err.Error())
						return
					}
					assert.NoError(t, err)
					assert.NotNil(t, res)
				})
			}

			backupService.AssertExpectations(t)
		})
	})
}

func TestScheduledBackups(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
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
		backupService := &mockBackupService{}
		mockedPbmPITRService := &mockPbmPITRService{}
		schedulerService := scheduler.New(db, backupService)
		backupSvc := NewBackupsService(db, backupService, nil, schedulerService, nil, mockedPbmPITRService)

		agent := setup(t, db.Querier, models.MySQLServiceType, t.Name(), "cluster")

		t.Run("schedule/change", func(t *testing.T) {
			req := &backupv1.ScheduleBackupRequest{
				ServiceId:      pointer.GetString(agent.ServiceID),
				LocationId:     locationRes.ID,
				CronExpression: "1 * * * *",
				StartTime:      timestamppb.New(time.Now()),
				Name:           "schedule_change",
				Description:    t.Name(),
				Enabled:        true,
				Mode:           backupv1.BackupMode_BACKUP_MODE_SNAPSHOT,
				DataModel:      backupv1.DataModel_DATA_MODEL_PHYSICAL,
				Retries:        maxRetriesAttempts - 1,
				RetryInterval:  durationpb.New(maxRetryInterval),
			}
			res, err := backupSvc.ScheduleBackup(ctx, req)

			require.NoError(t, err)
			require.NotEmpty(t, res.ScheduledBackupId)

			task, err := models.FindScheduledTaskByID(db.Querier, res.ScheduledBackupId)
			require.NoError(t, err)
			assert.Equal(t, models.ScheduledMySQLBackupTask, task.Type)
			assert.Equal(t, req.CronExpression, task.CronExpression)
			data := task.Data.MySQLBackupTask
			assert.Equal(t, req.Name, data.Name)
			assert.Equal(t, req.Description, data.Description)
			assert.Equal(t, req.ServiceId, data.ServiceID)
			assert.Equal(t, req.LocationId, data.LocationID)
			assert.Equal(t, req.Retries, data.Retries)
			assert.Equal(t, req.RetryInterval.AsDuration(), data.RetryInterval)

			changeReq := &backupv1.ChangeScheduledBackupRequest{
				ScheduledBackupId: task.ID,
				Enabled:           pointer.ToBool(false),
				CronExpression:    pointer.ToString("2 * * * *"),
				StartTime:         timestamppb.New(time.Now()),
				Name:              pointer.ToString("test"),
				Description:       pointer.ToString("test"),
				Retries:           pointer.ToUint32(0),
				RetryInterval:     durationpb.New(time.Second),
			}
			_, err = backupSvc.ChangeScheduledBackup(ctx, changeReq)

			assert.NoError(t, err)
			task, err = models.FindScheduledTaskByID(db.Querier, res.ScheduledBackupId)
			require.NoError(t, err)
			data = task.Data.MySQLBackupTask
			assert.Equal(t, *changeReq.CronExpression, task.CronExpression)
			assert.Equal(t, *changeReq.Enabled, !task.Disabled)
			assert.Equal(t, *changeReq.Name, data.Name)
			assert.Equal(t, *changeReq.Description, data.Description)
			assert.Equal(t, *changeReq.Retries, data.Retries)
			assert.Equal(t, changeReq.RetryInterval.AsDuration(), data.RetryInterval)
		})

		t.Run("list", func(t *testing.T) {
			res, err := backupSvc.ListScheduledBackups(ctx, &backupv1.ListScheduledBackupsRequest{})

			assert.NoError(t, err)
			assert.Len(t, res.ScheduledBackups, 1)
		})

		t.Run("remove", func(t *testing.T) {
			task, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
				CronExpression: "* * * * *",
				Type:           models.ScheduledMySQLBackupTask,
				Data:           &models.ScheduledTaskData{MySQLBackupTask: &models.MySQLBackupTaskData{CommonBackupTaskData: models.CommonBackupTaskData{Name: t.Name()}}},
			})
			require.NoError(t, err)

			id := task.ID

			_, err = models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:       "artifact",
				Vendor:     "mysql",
				LocationID: locationRes.ID,
				ServiceID:  *agent.ServiceID,
				DataModel:  models.PhysicalDataModel,
				Mode:       models.Snapshot,
				Status:     models.PendingBackupStatus,
				ScheduleID: id,
			})
			require.NoError(t, err)

			_, err = backupSvc.RemoveScheduledBackup(ctx, &backupv1.RemoveScheduledBackupRequest{
				ScheduledBackupId: task.ID,
			})
			assert.NoError(t, err)

			task, err = models.FindScheduledTaskByID(db.Querier, task.ID)
			assert.Nil(t, task)
			require.ErrorIs(t, err, models.ErrNotFound)

			artifacts, err := models.FindArtifacts(db.Querier, models.ArtifactFilters{
				ScheduleID: id,
			})

			assert.NoError(t, err)
			assert.Len(t, artifacts, 0)
		})
	})

	t.Run("mongo", func(t *testing.T) {
		agent := setup(t, db.Querier, models.MongoDBServiceType, t.Name(), "cluster")

		t.Run("PITR unsupported for physical model", func(t *testing.T) {
			ctx := context.Background()
			schedulerService := &mockScheduleService{}
			mockedPbmPITRService := &mockPbmPITRService{}
			backupSvc := NewBackupsService(db, nil, nil, schedulerService, nil, mockedPbmPITRService)

			schedulerService.On("Add", mock.Anything, mock.Anything).Return("", nil)
			_, err := backupSvc.ScheduleBackup(ctx, &backupv1.ScheduleBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    locationRes.ID,
				Name:          "name",
				Description:   "description",
				RetryInterval: durationpb.New(maxRetryInterval),
				Retries:       maxRetriesAttempts,
				DataModel:     backupv1.DataModel_DATA_MODEL_PHYSICAL,
				Mode:          backupv1.BackupMode_BACKUP_MODE_PITR,
			})
			require.Error(t, err)
			tests.AssertGRPCErrorRE(t, codes.InvalidArgument, "PITR is only supported for logical backups", err)
		})

		t.Run("normal", func(t *testing.T) {
			ctx := context.Background()
			schedulerService := &mockScheduleService{}
			mockedPbmPITRService := &mockPbmPITRService{}
			backupSvc := NewBackupsService(db, nil, nil, schedulerService, nil, mockedPbmPITRService)
			schedulerService.On("Add", mock.Anything, mock.Anything).Return(&models.ScheduledTask{}, nil)
			_, err := backupSvc.ScheduleBackup(ctx, &backupv1.ScheduleBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    locationRes.ID,
				Name:          "name",
				Description:   "description",
				RetryInterval: durationpb.New(maxRetryInterval),
				Retries:       maxRetriesAttempts,
				DataModel:     backupv1.DataModel_DATA_MODEL_PHYSICAL,
				Mode:          backupv1.BackupMode_BACKUP_MODE_SNAPSHOT,
			})
			require.NoError(t, err)
		})
	})
}

func TestGetLogs(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupService := &mockBackupService{}
	schedulerService := &mockScheduleService{}
	mockedPbmPITRService := &mockPbmPITRService{}
	backupSvc := NewBackupsService(db, backupService, nil, schedulerService, nil, mockedPbmPITRService)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	type testCase struct {
		offset uint32
		limit  uint32
		expect []uint32
	}
	testCases := []testCase{
		{
			expect: []uint32{0, 1, 2, 3, 4},
		},
		{
			offset: 3,
			expect: []uint32{3, 4},
		},
		{
			limit:  2,
			expect: []uint32{0, 1},
		},
		{
			offset: 1,
			limit:  3,
			expect: []uint32{1, 2, 3},
		},
		{
			offset: 5,
			expect: []uint32{},
		},
	}

	t.Run("get backup logs", func(t *testing.T) {
		artifactID := uuid.New().String()
		job, err := models.CreateJob(db.Querier, models.CreateJobParams{
			PMMAgentID: "agent",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBBackup: &models.MongoDBBackupJobData{
					ServiceID:  "svc",
					ArtifactID: artifactID,
				},
			},
		})
		require.NoError(t, err)
		for chunkID := 0; chunkID < 5; chunkID++ {
			_, err = models.CreateJobLog(db.Querier, models.CreateJobLogParams{
				JobID:   job.ID,
				ChunkID: chunkID,
				Data:    "not important",
			})
			assert.NoError(t, err)
		}

		for _, tc := range testCases {
			logs, err := backupSvc.GetLogs(ctx, &backupv1.GetLogsRequest{
				ArtifactId: artifactID,
				Offset:     tc.offset,
				Limit:      tc.limit,
			})
			assert.NoError(t, err)
			chunkIDs := make([]uint32, 0, len(logs.Logs))
			for _, log := range logs.Logs {
				chunkIDs = append(chunkIDs, log.ChunkId)
			}
			assert.Equal(t, tc.expect, chunkIDs)
		}
	})
}

func TestListPitrTimeranges(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
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

	backupService := &mockBackupService{}
	schedulerService := &mockScheduleService{}
	backupSvc := NewBackupsService(db, backupService, nil, schedulerService, nil, mockedPbmPITRService)

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
			Name:       "test_artifact",
			Vendor:     "test_vendor",
			LocationID: locationID,
			ServiceID:  "test_service",
			Mode:       models.PITR,
			DataModel:  models.LogicalDataModel,
			Status:     models.PendingBackupStatus,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, artifact.ID)

		response, err := backupSvc.ListPitrTimeranges(ctx, &backupv1.ListPitrTimerangesRequest{
			ArtifactId: artifact.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Timeranges, 1)
	})

	t.Run("fails for invalid artifact ID", func(t *testing.T) {
		unknownID := uuid.New().String()
		response, err := backupSvc.ListPitrTimeranges(ctx, &backupv1.ListPitrTimerangesRequest{
			ArtifactId: unknownID,
		})
		tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Artifact with ID '%s' not found.", unknownID)), err)
		assert.Nil(t, response)
	})

	t.Run("fails for non-PITR artifact", func(t *testing.T) {
		artifact, err := models.CreateArtifact(db.Querier, models.CreateArtifactParams{
			Name:       "test_non_pitr_artifact",
			Vendor:     "test_vendor",
			LocationID: locationID,
			ServiceID:  "test_service",
			Mode:       models.Snapshot,
			DataModel:  models.LogicalDataModel,
			Status:     models.PendingBackupStatus,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, artifact.ID)

		response, err := backupSvc.ListPitrTimeranges(ctx, &backupv1.ListPitrTimerangesRequest{
			ArtifactId: artifact.ID,
		})
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, "Artifact is not a PITR artifact."), err)
		assert.Nil(t, response)
	})
	mock.AssertExpectationsForObjects(t, mockedPbmPITRService)
}

func TestArtifactMetadataListToProto(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
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
		Name:       "test_artifact",
		Vendor:     "test_vendor",
		LocationID: loc.ID,
		ServiceID:  "test_service",
		Mode:       models.PITR,
		DataModel:  models.LogicalDataModel,
		Status:     models.PendingBackupStatus,
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

	expected := []*backupv1.Metadata{
		{
			FileList: []*backupv1.File{
				{Name: "dir1", IsDirectory: true},
				{Name: "file1"},
				{Name: "file2"},
				{Name: "file3"},
			},
		},
		{
			FileList: []*backupv1.File{
				{Name: "dir2", IsDirectory: true},
				{Name: "file4"},
				{Name: "file5"},
				{Name: "file6"},
			},
			RestoreTo:          &timestamppb.Timestamp{Seconds: 123, Nanos: 456},
			BackupToolMetadata: &backupv1.Metadata_PbmMetadata{PbmMetadata: &backupv1.PbmMetadata{Name: "backup tool data name"}},
		},
	}

	actual := artifactMetadataListToProto(artifact)

	assert.Equal(t, expected, actual)
}
