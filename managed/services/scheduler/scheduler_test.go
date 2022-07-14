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

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestService(t *testing.T) {
	setup := func(t *testing.T, ctx context.Context) *Service {
		t.Helper()
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		backupService := &mockBackupService{}
		svc := New(db, backupService)

		go svc.Run(ctx)
		for !svc.scheduler.IsRunning() {
			// Wait a while, so scheduler is running
			time.Sleep(time.Millisecond * 10)
		}

		return svc
	}

	t.Run("invalid cron expression", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		svc := setup(t, ctx)

		task, err := NewMongoDBBackupTask(&BackupTaskParams{
			ServiceID:     "/service/test",
			LocationID:    "/location/test",
			Name:          "test",
			Description:   "test backup task",
			DataModel:     models.LogicalDataModel,
			Mode:          models.Snapshot,
			Retention:     7,
			Retries:       3,
			RetryInterval: 5 * time.Second,
		})
		require.NoError(t, err)

		cronExpr := "invalid * cron * expression"
		startAt := time.Now().Truncate(time.Second).UTC()
		_, err = svc.Add(task, AddParams{
			CronExpression: cronExpr,
			StartAt:        startAt,
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Invalid cron expression: failed to parse int from invalid: strconv.Atoi: parsing \"invalid\": invalid syntax"), err)
	})

	t.Run("normal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		svc := setup(t, ctx)

		task, err := NewMongoDBBackupTask(&BackupTaskParams{
			ServiceID:     "/service/test",
			LocationID:    "/location/test",
			Name:          "test",
			Description:   "test backup task",
			DataModel:     models.LogicalDataModel,
			Mode:          models.Snapshot,
			Retention:     7,
			Retries:       3,
			RetryInterval: 5 * time.Second,
		})
		require.NoError(t, err)

		cronExpr := "* * * * *"
		startAt := time.Now().Truncate(time.Second).UTC()
		dbTask, err := svc.Add(task, AddParams{
			CronExpression: cronExpr,
			StartAt:        startAt,
		})
		require.NoError(t, err)
		assert.Len(t, svc.scheduler.Jobs(), 1)

		findJob, err := models.FindScheduledTaskByID(svc.db.Querier, dbTask.ID)
		require.NoError(t, err)

		assert.Equal(t, startAt, dbTask.StartAt)
		assert.Equal(t, cronExpr, findJob.CronExpression)
		assert.Truef(t, dbTask.NextRun.After(startAt), "next run %s is before startAt %s", dbTask.NextRun, startAt)

		err = svc.Remove(dbTask.ID)
		require.NoError(t, err)
		assert.Len(t, svc.scheduler.Jobs(), 0)

		_, err = models.FindScheduledTaskByID(svc.db.Querier, dbTask.ID)
		tests.AssertGRPCError(t, status.Newf(codes.NotFound, `ScheduledTask with ID "%s" not found.`, dbTask.ID), err)
	})
}
