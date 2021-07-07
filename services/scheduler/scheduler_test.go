// pmm-managed
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func setup(t *testing.T) *Service {
	t.Helper()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	return New(db)
}
func TestService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := setup(t)
	go func() {
		svc.Run(ctx)
	}()
	for !svc.scheduler.IsRunning() {
		// Wait a while, so scheduler is running
		time.Sleep(time.Millisecond * 10)
	}

	task := NewPrintTask("test")
	cronExpr := "* * * * *"
	startAt := time.Now().Truncate(time.Second).UTC()
	dbTask, err := svc.Add(task, AddParams{
		CronExpression: cronExpr,
		StartAt:        startAt,
	})
	assert.NoError(t, err)

	assert.Len(t, svc.scheduler.Jobs(), 1)
	findJob, err := models.FindScheduledTaskByID(svc.db.Querier, dbTask.ID)
	assert.NoError(t, err)

	assert.Equal(t, startAt, dbTask.StartAt)
	assert.Equal(t, cronExpr, findJob.CronExpression)
	assert.Truef(t, dbTask.NextRun.After(startAt), "next run %s is before startAt %s", dbTask.NextRun, startAt)

	err = svc.Remove(dbTask.ID)
	assert.NoError(t, err)
	assert.Len(t, svc.scheduler.Jobs(), 0)
	_, err = models.FindScheduledTaskByID(svc.db.Querier, dbTask.ID)
	tests.AssertGRPCError(t, status.Newf(codes.NotFound, `ScheduledTask with ID "%s" not found.`, dbTask.ID), err)

}
