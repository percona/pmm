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

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
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
	setup := func(t *testing.T, ctx context.Context, serviceType models.ServiceType, serviceName string) (*Service, *models.Service, *models.BackupLocation) {
		t.Helper()
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "test-node-" + t.Name(),
		})
		require.NoError(t, err)

		service, err := models.AddNewService(db.Querier, serviceType, &models.AddDBMSServiceParams{
			ServiceName: serviceName,
			NodeID:      node.NodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(60000),
		})
		require.NoError(t, err)

		location, err := models.CreateBackupLocation(db.Querier, models.CreateBackupLocationParams{
			Name: "test_location",
			BackupLocationConfig: models.BackupLocationConfig{
				FilesystemConfig: &models.FilesystemLocationConfig{
					Path: "/tmp",
				},
			},
		})
		require.NoError(t, err)

		backupService := &mockBackupService{}
		schedulerSvc := New(db, backupService)

		go schedulerSvc.Run(ctx)
		for !schedulerSvc.scheduler.IsRunning() {
			// Wait a while, so scheduler is running
			time.Sleep(time.Millisecond * 10)
		}

		return schedulerSvc, service, location
	}

	t.Run("invalid cron expression", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		scheduler, service, location := setup(t, ctx, models.MongoDBServiceType, "mongo_service")

		task, err := NewMongoDBBackupTask(&BackupTaskParams{
			ServiceID:     service.ServiceID,
			LocationID:    location.ID,
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
		_, err = scheduler.Add(task, AddParams{
			CronExpression: cronExpr,
			StartAt:        startAt,
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Invalid cron expression: failed to parse int from invalid: strconv.Atoi: parsing \"invalid\": invalid syntax"), err)
	})

	t.Run("normal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		scheduler, service, location := setup(t, ctx, models.MongoDBServiceType, "mongo_service")

		task, err := NewMongoDBBackupTask(&BackupTaskParams{
			ServiceID:     service.ServiceID,
			LocationID:    location.ID,
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
		dbTask, err := scheduler.Add(task, AddParams{
			CronExpression: cronExpr,
			StartAt:        startAt,
		})
		require.NoError(t, err)
		assert.Len(t, scheduler.scheduler.Jobs(), 1)

		findJob, err := models.FindScheduledTaskByID(scheduler.db.Querier, dbTask.ID)
		require.NoError(t, err)

		assert.Equal(t, startAt, dbTask.StartAt)
		assert.Equal(t, cronExpr, findJob.CronExpression)
		assert.Truef(t, dbTask.NextRun.After(startAt), "next run %s is before startAt %s", dbTask.NextRun, startAt)

		err = scheduler.Remove(dbTask.ID)
		require.NoError(t, err)
		assert.Empty(t, scheduler.scheduler.Jobs())

		_, err = models.FindScheduledTaskByID(scheduler.db.Querier, dbTask.ID)
		assert.ErrorIs(t, err, models.ErrNotFound)
	})
}
