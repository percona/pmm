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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/scheduler"
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
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		backupSvc := NewBackupsService(db, backupService, nil, nil)
		agent := setup(t, db.Querier, models.MySQLServiceType, t.Name(), "cluster")

		for _, tc := range []struct {
			testName    string
			backupError error
			code        backuppb.ErrorCode
		}{
			{
				testName:    "xtrabackup not installed",
				backupError: backup.ErrXtrabackupNotInstalled,
				code:        backuppb.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED,
			},
			{
				testName:    "invalid xtrabackup",
				backupError: backup.ErrInvalidXtrabackup,
				code:        backuppb.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP,
			},
			{
				testName:    "incompatible xtrabackup",
				backupError: backup.ErrIncompatibleXtrabackup,
				code:        backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP,
			},
		} {
			t.Run(tc.testName, func(t *testing.T) {
				backupError := fmt.Errorf("error: %w", tc.backupError)
				backupService.On("PerformBackup", mock.Anything, mock.Anything).
					Return("", backupError).Once()
				ctx := context.Background()
				resp, err := backupSvc.StartBackup(ctx, &backuppb.StartBackupRequest{
					ServiceId:     *agent.ServiceID,
					LocationId:    "locationID",
					Name:          "name",
					Description:   "description",
					DataModel:     backuppb.DataModel_PHYSICAL,
					RetryInterval: nil,
					Retries:       0,
					Compression:   backuppb.BackupCompression_ZSTD,
				})
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.FailedPrecondition, st.Code())
				assert.Equal(t, backupError.Error(), st.Message())
				require.Len(t, st.Details(), 1)
				detailedError, ok := st.Details()[0].(*backuppb.Error)
				require.True(t, ok)
				assert.Equal(t, tc.code, detailedError.Code)
			})
		}

		t.Run("compression test cases", func(t *testing.T) {
			compressionTests := []struct {
				name        string
				compression backuppb.BackupCompression
				shouldError bool
			}{
				{
					name:        "QuickLZ compression",
					compression: backuppb.BackupCompression_QUICKLZ,
					shouldError: false,
				},
				{
					name:        "ZSTD compression",
					compression: backuppb.BackupCompression_ZSTD,
					shouldError: false,
				},
				{
					name:        "LZ4 compression",
					compression: backuppb.BackupCompression_LZ4,
					shouldError: false,
				},
				{
					name:        "None compression",
					compression: backuppb.BackupCompression_NONE,
					shouldError: false,
				},
			}

			for _, tc := range compressionTests {
				t.Run(tc.name, func(t *testing.T) {
					backupService.On("PerformBackup", mock.Anything, mock.Anything).
						Return("artifact_id", nil).Once()
					ctx := context.Background()
					resp, err := backupSvc.StartBackup(ctx, &backuppb.StartBackupRequest{
						ServiceId:     *agent.ServiceID,
						LocationId:    "locationID",
						Name:          "name",
						Description:   "description",
						DataModel:     backuppb.DataModel_PHYSICAL,
						RetryInterval: nil,
						Retries:       0,
						Compression:   tc.compression,
					})
					if tc.shouldError {
						assert.Error(t, err)
						assert.Nil(t, resp)
					} else {
						assert.NoError(t, err)
						assert.NotNil(t, resp)
						assert.Equal(t, "artifact_id", resp.ArtifactId)
					}
				})
			}
		})
	})

	t.Run("mongodb", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
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
			backupSvc := NewBackupsService(db, backupService, nil, nil)
			backupService.On("PerformBackup", mock.Anything, mock.Anything).Return("", nil)
			_, err := backupSvc.StartBackup(ctx, &backuppb.StartBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    locationRes.ID,
				Name:          "name",
				Description:   "description",
				RetryInterval: nil,
				Retries:       0,
				DataModel:     backuppb.DataModel_PHYSICAL,
				Compression:   backuppb.BackupCompression_S2,
			})
			require.NoError(t, err)
		})

		t.Run("mongodb compression test cases", func(t *testing.T) {
			compressionTests := []struct {
				name        string
				compression backuppb.BackupCompression
				shouldError bool
			}{
				{
					name:        "GZIP compression",
					compression: backuppb.BackupCompression_GZIP,
					shouldError: false,
				},
				{
					name:        "Snappy compression",
					compression: backuppb.BackupCompression_SNAPPY,
					shouldError: false,
				},
				{
					name:        "LZ4 compression",
					compression: backuppb.BackupCompression_LZ4,
					shouldError: false,
				},
				{
					name:        "S2 compression",
					compression: backuppb.BackupCompression_S2,
					shouldError: false,
				},
				{
					name:        "PGZIP compression",
					compression: backuppb.BackupCompression_PGZIP,
					shouldError: false,
				},
				{
					name:        "ZSTD compression",
					compression: backuppb.BackupCompression_ZSTD,
					shouldError: false,
				},
				{
					name:        "None compression",
					compression: backuppb.BackupCompression_NONE,
					shouldError: false,
				},
			}

			for _, tc := range compressionTests {
				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()
					backupService := &mockBackupService{}
					backupSvc := NewBackupsService(db, backupService, nil, nil)
					backupService.On("PerformBackup", mock.Anything, mock.Anything).Return("artifact_id", nil)
					_, err := backupSvc.StartBackup(ctx, &backuppb.StartBackupRequest{
						ServiceId:     *agent.ServiceID,
						LocationId:    locationRes.ID,
						Name:          "name",
						Description:   "description",
						RetryInterval: nil,
						Retries:       0,
						DataModel:     backuppb.DataModel_PHYSICAL,
						Compression:   tc.compression,
					})
					if tc.shouldError {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
					}
				})
			}
		})

		t.Run("check folder and artifact name", func(t *testing.T) {
			ctx := context.Background()
			backupService := &mockBackupService{}
			backupSvc := NewBackupsService(db, backupService, nil, nil)

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
					res, err := backupSvc.StartBackup(ctx, &backuppb.StartBackupRequest{
						Name:        test.BackupName,
						Folder:      test.Folder,
						ServiceId:   *agent.ServiceID,
						DataModel:   backuppb.DataModel_LOGICAL,
						Compression: backuppb.BackupCompression_S2,
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

func TestRestoreBackupErrors(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupService := &mockBackupService{}
	backupSvc := NewBackupsService(db, backupService, nil, nil)

	for _, tc := range []struct {
		testName    string
		backupError error
		code        backuppb.ErrorCode
	}{
		{
			testName:    "xtrabackup not installed",
			backupError: backup.ErrXtrabackupNotInstalled,
			code:        backuppb.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED,
		},
		{
			testName:    "invalid xtrabackup",
			backupError: backup.ErrInvalidXtrabackup,
			code:        backuppb.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP,
		},
		{
			testName:    "incompatible xtrabackup",
			backupError: backup.ErrIncompatibleXtrabackup,
			code:        backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP,
		},
		{
			testName:    "target MySQL is not compatible",
			backupError: backup.ErrIncompatibleTargetMySQL,
			code:        backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MYSQL,
		},
		{
			testName:    "target MongoDB is not compatible",
			backupError: backup.ErrIncompatibleTargetMongoDB,
			code:        backuppb.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MONGODB,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			backupError := fmt.Errorf("error: %w", tc.backupError)
			backupService.On("RestoreBackup", mock.Anything, "serviceID1", "artifactID1", mock.Anything).
				Return("", backupError).Once()
			ctx := context.Background()
			resp, err := backupSvc.RestoreBackup(ctx, &backuppb.RestoreBackupRequest{
				ServiceId:  "serviceID1",
				ArtifactId: "artifactID1",
			})
			assert.Nil(t, resp)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.FailedPrecondition, st.Code())
			assert.Equal(t, backupError.Error(), st.Message())
			require.Len(t, st.Details(), 1)
			detailedError, ok := st.Details()[0].(*backuppb.Error)
			require.True(t, ok)
			assert.Equal(t, tc.code, detailedError.Code)
		})
	}
}

func TestScheduledBackups(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
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
		schedulerService := scheduler.New(db, backupService)
		backupSvc := NewBackupsService(db, backupService, nil, schedulerService)

		agent := setup(t, db.Querier, models.MySQLServiceType, t.Name(), "cluster")

		t.Run("schedule/change", func(t *testing.T) {
			req := &backuppb.ScheduleBackupRequest{
				ServiceId:      pointer.GetString(agent.ServiceID),
				LocationId:     locationRes.ID,
				CronExpression: "1 * * * *",
				StartTime:      timestamppb.New(time.Now()),
				Name:           "schedule_change",
				Description:    t.Name(),
				Enabled:        true,
				Mode:           backuppb.BackupMode_SNAPSHOT,
				DataModel:      backuppb.DataModel_PHYSICAL,
				Compression:    backuppb.BackupCompression_ZSTD,
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

			changeReq := &backuppb.ChangeScheduledBackupRequest{
				ScheduledBackupId: task.ID,
				Enabled:           wrapperspb.Bool(false),
				CronExpression:    wrapperspb.String("2 * * * *"),
				StartTime:         timestamppb.New(time.Now()),
				Name:              wrapperspb.String("test"),
				Description:       wrapperspb.String("test"),
				Retries:           wrapperspb.UInt32(0),
				RetryInterval:     durationpb.New(time.Second),
			}
			_, err = backupSvc.ChangeScheduledBackup(ctx, changeReq)

			assert.NoError(t, err)
			task, err = models.FindScheduledTaskByID(db.Querier, res.ScheduledBackupId)
			require.NoError(t, err)
			data = task.Data.MySQLBackupTask
			assert.Equal(t, changeReq.CronExpression.GetValue(), task.CronExpression)
			assert.Equal(t, changeReq.Enabled.GetValue(), !task.Disabled)
			assert.Equal(t, changeReq.Name.GetValue(), data.Name)
			assert.Equal(t, changeReq.Description.GetValue(), data.Description)
			assert.Equal(t, changeReq.Retries.GetValue(), data.Retries)
			assert.Equal(t, changeReq.RetryInterval.AsDuration(), data.RetryInterval)
		})

		t.Run("list", func(t *testing.T) {
			res, err := backupSvc.ListScheduledBackups(ctx, &backuppb.ListScheduledBackupsRequest{})

			assert.NoError(t, err)
			assert.Len(t, res.ScheduledBackups, 1)
		})

		t.Run("remove", func(t *testing.T) {
			task, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
				CronExpression: "* * * * *",
				Type:           models.ScheduledMySQLBackupTask,
				Data:           &models.ScheduledTaskData{MySQLBackupTask: &models.MySQLBackupTaskData{CommonBackupTaskData: models.CommonBackupTaskData{Name: t.Name(), Compression: models.ZSTD}}},
			})
			require.NoError(t, err)

			id := task.ID

			_, err = models.CreateArtifact(db.Querier, models.CreateArtifactParams{
				Name:        "artifact",
				Vendor:      "mysql",
				LocationID:  locationRes.ID,
				ServiceID:   *agent.ServiceID,
				DataModel:   models.PhysicalDataModel,
				Mode:        models.Snapshot,
				Status:      models.PendingBackupStatus,
				ScheduleID:  id,
				Compression: models.ZSTD,
			})
			require.NoError(t, err)

			_, err = backupSvc.RemoveScheduledBackup(ctx, &backuppb.RemoveScheduledBackupRequest{
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
			backupSvc := NewBackupsService(db, nil, nil, schedulerService)

			schedulerService.On("Add", mock.Anything, mock.Anything).Return("", nil)
			_, err := backupSvc.ScheduleBackup(ctx, &backuppb.ScheduleBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    locationRes.ID,
				Name:          "name",
				Description:   "description",
				RetryInterval: durationpb.New(maxRetryInterval),
				Retries:       maxRetriesAttempts,
				DataModel:     backuppb.DataModel_PHYSICAL,
				Mode:          backuppb.BackupMode_PITR,
				Compression:   backuppb.BackupCompression_S2,
			})
			require.Error(t, err)
			tests.AssertGRPCErrorRE(t, codes.InvalidArgument, "PITR is only supported for logical backups", err)
		})

		t.Run("normal", func(t *testing.T) {
			ctx := context.Background()
			schedulerService := &mockScheduleService{}
			backupSvc := NewBackupsService(db, nil, nil, schedulerService)
			schedulerService.On("Add", mock.Anything, mock.Anything).Return(&models.ScheduledTask{}, nil)
			_, err := backupSvc.ScheduleBackup(ctx, &backuppb.ScheduleBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    locationRes.ID,
				Name:          "name",
				Description:   "description",
				RetryInterval: durationpb.New(maxRetryInterval),
				Retries:       maxRetriesAttempts,
				DataModel:     backuppb.DataModel_PHYSICAL,
				Mode:          backuppb.BackupMode_SNAPSHOT,
				Compression:   backuppb.BackupCompression_S2,
			})
			require.NoError(t, err)
		})
	})
}

func TestGetLogs(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupService := &mockBackupService{}
	schedulerService := &mockScheduleService{}
	backupSvc := NewBackupsService(db, backupService, nil, schedulerService)
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
		job, err := models.CreateJob(db.Querier, models.CreateJobParams{
			PMMAgentID: "agent",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBBackup: &models.MongoDBBackupJobData{
					ServiceID:  "svc",
					ArtifactID: "artifact",
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
			logs, err := backupSvc.GetLogs(ctx, &backuppb.GetLogsRequest{
				ArtifactId: "artifact",
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

	t.Run("get physical restore logs", func(t *testing.T) {
		job, err := models.CreateJob(db.Querier, models.CreateJobParams{
			PMMAgentID: "agent",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBRestoreBackup: &models.MongoDBRestoreBackupJobData{
					ServiceID: "svc",
					RestoreID: "physical-restore-1",
					DataModel: models.PhysicalDataModel,
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
			logs, err := backupSvc.GetLogs(ctx, &backuppb.GetLogsRequest{
				RestoreId: "physical-restore-1",
				Offset:    tc.offset,
				Limit:     tc.limit,
			})
			assert.NoError(t, err)
			chunkIDs := make([]uint32, 0, len(logs.Logs))
			for _, log := range logs.Logs {
				chunkIDs = append(chunkIDs, log.ChunkId)
			}
			assert.Equal(t, tc.expect, chunkIDs)
		}
	})

	t.Run("get logical restore logs", func(t *testing.T) {
		logicalRestore, err := models.CreateJob(db.Querier, models.CreateJobParams{
			PMMAgentID: "agent",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBRestoreBackup: &models.MongoDBRestoreBackupJobData{
					ServiceID: "svc",
					RestoreID: "logical-restore-1",
					DataModel: models.LogicalDataModel,
				},
			},
		})
		require.NoError(t, err)
		for chunkID := 0; chunkID < 5; chunkID++ {
			_, err = models.CreateJobLog(db.Querier, models.CreateJobLogParams{
				JobID:   logicalRestore.ID,
				ChunkID: chunkID,
				Data:    "not important",
			})
			assert.NoError(t, err)
		}

		for _, tc := range testCases {
			logs, err := backupSvc.GetLogs(ctx, &backuppb.GetLogsRequest{
				RestoreId: "logical-restore-1",
				Offset:    tc.offset,
				Limit:     tc.limit,
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

func TestConvertCompressionToBackupCompression(t *testing.T) {
	tests := []struct {
		name        string
		compression backuppb.BackupCompression
		expected    models.BackupCompression
		shouldError bool
	}{
		{
			name:        "QuickLZ compression",
			compression: backuppb.BackupCompression_QUICKLZ,
			expected:    models.QuickLZ,
			shouldError: false,
		},
		{
			name:        "ZSTD compression",
			compression: backuppb.BackupCompression_ZSTD,
			expected:    models.ZSTD,
			shouldError: false,
		},
		{
			name:        "LZ4 compression",
			compression: backuppb.BackupCompression_LZ4,
			expected:    models.LZ4,
			shouldError: false,
		},
		{
			name:        "S2 compression",
			compression: backuppb.BackupCompression_S2,
			expected:    models.S2,
			shouldError: false,
		},
		{
			name:        "GZIP compression",
			compression: backuppb.BackupCompression_GZIP,
			expected:    models.GZIP,
			shouldError: false,
		},
		{
			name:        "Snappy compression",
			compression: backuppb.BackupCompression_SNAPPY,
			expected:    models.Snappy,
			shouldError: false,
		},
		{
			name:        "PGZIP compression",
			compression: backuppb.BackupCompression_PGZIP,
			expected:    models.PGZIP,
			shouldError: false,
		},
		{
			name:        "None compression",
			compression: backuppb.BackupCompression_NONE,
			expected:    models.None,
			shouldError: false,
		},
		{
			name:        "invalid compression",
			compression: backuppb.BackupCompression_BACKUP_COMPRESSION_INVALID,
			expected:    models.BackupCompression(""),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertCompressionToBackupCompression(tt.compression)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Equal(t, models.BackupCompression(""), result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
