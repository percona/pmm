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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	backupv1 "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRestoreServiceGetLogs(t *testing.T) {
	ctx := context.Background()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	restoreSvc := NewRestoreService(db)

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
		restoreID := models.NormalizeRestoreID(uuid.New().String())
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
		restoreID := models.NormalizeRestoreID(uuid.New().String())
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
