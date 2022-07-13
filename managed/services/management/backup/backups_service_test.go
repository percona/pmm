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

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/services/scheduler"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func setup(t *testing.T, q *reform.Querier, serviceName string) *models.Agent {
	t.Helper()
	node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "test-node",
	})
	require.NoError(t, err)

	pmmAgent, err := models.CreatePMMAgent(q, node.NodeID, nil)
	require.NoError(t, err)
	require.NoError(t, q.Update(pmmAgent))

	mysql, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: serviceName,
		NodeID:      node.NodeID,
		Address:     pointer.ToString("127.0.0.1"),
		Port:        pointer.ToUint16(3306),
	})
	require.NoError(t, err)

	agent, err := models.CreateAgent(q, models.MySQLdExporterType, &models.CreateAgentParams{
		PMMAgentID: pmmAgent.AgentID,
		ServiceID:  mysql.ServiceID,
		Username:   "user",
		Password:   "password",
	})
	require.NoError(t, err)
	return agent
}

func TestStartBackupErrors(t *testing.T) {
	backupService := &mockBackupService{}
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupSvc := NewBackupsService(db, backupService, nil)
	agent := setup(t, db.Querier, t.Name())

	for _, tc := range []struct {
		testName    string
		backupError error
		code        backupv1beta1.ErrorCode
	}{
		{
			testName:    "xtrabackup not installed",
			backupError: backup.ErrXtrabackupNotInstalled,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED,
		},
		{
			testName:    "invalid xtrabackup",
			backupError: backup.ErrInvalidXtrabackup,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP,
		},
		{
			testName:    "incompatible xtrabackup",
			backupError: backup.ErrIncompatibleXtrabackup,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			backupError := fmt.Errorf("error: %w", tc.backupError)
			backupService.On("PerformBackup", mock.Anything, mock.Anything).
				Return("", backupError).Once()
			ctx := context.Background()
			resp, err := backupSvc.StartBackup(ctx, &backupv1beta1.StartBackupRequest{
				ServiceId:     *agent.ServiceID,
				LocationId:    "locationID",
				Name:          "name",
				Description:   "description",
				RetryInterval: nil,
				Retries:       0,
			})
			assert.Nil(t, resp)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.FailedPrecondition, st.Code())
			assert.Equal(t, backupError.Error(), st.Message())
			require.Len(t, st.Details(), 1)
			detailedError, ok := st.Details()[0].(*backupv1beta1.Error)
			require.True(t, ok)
			assert.Equal(t, tc.code, detailedError.Code)
		})
	}
}

func TestRestoreBackupErrors(t *testing.T) {
	backupService := &mockBackupService{}
	backupSvc := NewBackupsService(nil, backupService, nil)

	for _, tc := range []struct {
		testName    string
		backupError error
		code        backupv1beta1.ErrorCode
	}{
		{
			testName:    "xtrabackup not installed",
			backupError: backup.ErrXtrabackupNotInstalled,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_XTRABACKUP_NOT_INSTALLED,
		},
		{
			testName:    "invalid xtrabackup",
			backupError: backup.ErrInvalidXtrabackup,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_INVALID_XTRABACKUP,
		},
		{
			testName:    "incompatible xtrabackup",
			backupError: backup.ErrIncompatibleXtrabackup,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_INCOMPATIBLE_XTRABACKUP,
		},
		{
			testName:    "target MySQL is not compatible",
			backupError: backup.ErrIncompatibleTargetMySQL,
			code:        backupv1beta1.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MYSQL,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			backupError := fmt.Errorf("error: %w", tc.backupError)
			backupService.On("RestoreBackup", mock.Anything, "serviceID1", "artifactID1").
				Return("", backupError).Once()
			ctx := context.Background()
			resp, err := backupSvc.RestoreBackup(ctx, &backupv1beta1.RestoreBackupRequest{
				ServiceId:  "serviceID1",
				ArtifactId: "artifactID1",
			})
			assert.Nil(t, resp)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.FailedPrecondition, st.Code())
			assert.Equal(t, backupError.Error(), st.Message())
			require.Len(t, st.Details(), 1)
			detailedError, ok := st.Details()[0].(*backupv1beta1.Error)
			require.True(t, ok)
			assert.Equal(t, tc.code, detailedError.Code)
		})
	}
}

func TestScheduledBackups(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	backupService := &mockBackupService{}
	backupService.On("SwitchMongoPITR", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	schedulerService := scheduler.New(db, backupService)
	backupSvc := NewBackupsService(db, backupService, schedulerService)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	agent := setup(t, db.Querier, t.Name())
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

	t.Run("schedule/change", func(t *testing.T) {
		req := &backupv1beta1.ScheduleBackupRequest{
			ServiceId:      pointer.GetString(agent.ServiceID),
			LocationId:     locationRes.ID,
			CronExpression: "1 * * * *",
			StartTime:      timestamppb.New(time.Now()),
			Name:           t.Name(),
			Description:    t.Name(),
			Enabled:        true,
			Mode:           backupv1beta1.BackupMode_SNAPSHOT,
			Retries:        maxRetriesAttempts - 1,
			RetryInterval:  durationpb.New(maxRetryInterval),
		}
		res, err := backupSvc.ScheduleBackup(ctx, req)

		assert.NoError(t, err)
		assert.NotEmpty(t, res.ScheduledBackupId)

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

		changeReq := &backupv1beta1.ChangeScheduledBackupRequest{
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
		res, err := backupSvc.ListScheduledBackups(ctx, &backupv1beta1.ListScheduledBackupsRequest{})

		assert.NoError(t, err)
		assert.Len(t, res.ScheduledBackups, 1)
	})

	t.Run("remove", func(t *testing.T) {
		task, err := models.CreateScheduledTask(db.Querier, models.CreateScheduledTaskParams{
			CronExpression: "* * * * *",
			Type:           models.ScheduledMySQLBackupTask,
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

		_, err = backupSvc.RemoveScheduledBackup(ctx, &backupv1beta1.RemoveScheduledBackupRequest{
			ScheduledBackupId: task.ID,
		})
		assert.NoError(t, err)

		task, err = models.FindScheduledTaskByID(db.Querier, task.ID)
		assert.Nil(t, task)
		tests.AssertGRPCError(t, status.Newf(codes.NotFound, `ScheduledTask with ID "%s" not found.`, id), err)

		artifacts, err := models.FindArtifacts(db.Querier, models.ArtifactFilters{
			ScheduleID: id,
		})

		assert.NoError(t, err)
		assert.Len(t, artifacts, 0)
	})
}

func TestGetLogs(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupService := &mockBackupService{}
	schedulerService := &mockScheduleService{}
	backupSvc := NewBackupsService(db, backupService, schedulerService)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

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
	for _, tc := range testCases {
		logs, err := backupSvc.GetLogs(ctx, &backupv1beta1.GetLogsRequest{
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
}
