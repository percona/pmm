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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	backupv1 "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRestoreServiceGetLogs(t *testing.T) {
	ctx := context.Background()

	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupService := &mockBackupService{}
	scheduleService := &mockScheduleService{}
	restoreSvc := NewRestoreService(db, backupService, scheduleService)

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

	t.Run("get physical restore logs", func(t *testing.T) {
		restoreID := uuid.New().String()
		job, err := models.CreateJob(db.Querier, models.CreateJobParams{
			PMMAgentID: "agent",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBRestoreBackup: &models.MongoDBRestoreBackupJobData{
					ServiceID: "svc",
					RestoreID: restoreID,
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
			logs, err := restoreSvc.GetLogs(ctx, &backupv1.RestoreServiceGetLogsRequest{
				RestoreId: restoreID,
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
		restoreID := uuid.New().String()
		logicalRestore, err := models.CreateJob(db.Querier, models.CreateJobParams{
			PMMAgentID: "agent",
			Type:       models.MongoDBBackupJob,
			Data: &models.JobData{
				MongoDBRestoreBackup: &models.MongoDBRestoreBackupJobData{
					ServiceID: "svc",
					RestoreID: restoreID,
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
			logs, err := restoreSvc.GetLogs(ctx, &backupv1.RestoreServiceGetLogsRequest{
				RestoreId: restoreID,
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

func TestRestoreBackupErrors(t *testing.T) {
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	backupService := &mockBackupService{}
	scheduleService := &mockScheduleService{}
	restoreSvc := NewRestoreService(db, backupService, scheduleService)

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
		{
			testName:    "target MySQL is not compatible",
			backupError: backup.ErrIncompatibleTargetMySQL,
			code:        backupv1.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MYSQL,
		},
		{
			testName:    "target MongoDB is not compatible",
			backupError: backup.ErrIncompatibleTargetMongoDB,
			code:        backupv1.ErrorCode_ERROR_CODE_INCOMPATIBLE_TARGET_MONGODB,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			backupError := fmt.Errorf("error: %w", tc.backupError)
			backupService.On("RestoreBackup", mock.Anything, "serviceID1", "artifactID1", mock.Anything).
				Return("", backupError).Once()
			ctx := context.Background()
			resp, err := restoreSvc.RestoreBackup(ctx, &backupv1.RestoreBackupRequest{
				ServiceId:  "serviceID1",
				ArtifactId: "artifactID1",
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
}
