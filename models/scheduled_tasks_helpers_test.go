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

package models_test

import (
	"sort"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestScheduledTaskHelpers(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.Rollback())
		require.NoError(t, sqlDB.Close())
	})

	createParams := models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMySQLBackupTask,
		Data: models.ScheduledTaskData{
			MySQLBackupTask: &models.MySQLBackupTaskData{
				ServiceID:   "",
				LocationID:  "",
				Name:        "task",
				Description: "",
			},
		},
		Disabled: false,
	}

	t.Run("CreateAndFind", func(t *testing.T) {
		task, err := models.CreateScheduledTask(tx.Querier, createParams)
		assert.NoError(t, err)

		task, err = models.FindScheduledTaskByID(tx.Querier, task.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, task.ID)
		assert.Equal(t, createParams.CronExpression, task.CronExpression)
		assert.Equal(t, createParams.Type, task.Type)
		assert.Equal(t, createParams.Disabled, task.Disabled)
		require.NotNil(t, task.Data.MySQLBackupTask)
		assert.Equal(t, createParams.Data.MySQLBackupTask.Name, task.Data.MySQLBackupTask.Name)

		_, err = models.CreateScheduledTask(tx.Querier, models.CreateScheduledTaskParams{
			CronExpression: "a * * * *",
			Type:           models.ScheduledMySQLBackupTask,
		})
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "Invalid cron expression")
	})

	t.Run("Change", func(t *testing.T) {
		task, err := models.CreateScheduledTask(tx.Querier, createParams)
		assert.NoError(t, err)

		changeParams := models.ChangeScheduledTaskParams{
			NextRun: pointer.ToTime(time.Now()),
			LastRun: pointer.ToTime(time.Now()),
			Disable: pointer.ToBool(true),
			Running: pointer.ToBool(true),
			Error:   pointer.ToString("something"),
		}
		task, err = models.ChangeScheduledTask(tx.Querier, task.ID, changeParams)
		assert.NoError(t, err)
		assert.Equal(t, *changeParams.NextRun, task.NextRun)
		assert.Equal(t, *changeParams.LastRun, task.LastRun)
		assert.Equal(t, *changeParams.Disable, task.Disabled)
		assert.Equal(t, *changeParams.Running, task.Running)
		assert.Equal(t, *changeParams.Error, task.Error)
	})

	t.Run("Remove", func(t *testing.T) {
		task, err := models.CreateScheduledTask(tx.Querier, createParams)
		assert.NoError(t, err)

		err = models.RemoveScheduledTask(tx.Querier, task.ID)
		assert.NoError(t, err)

		_, err = models.FindScheduledTaskByID(tx.Querier, task.ID)
		assert.Error(t, err, "task is not removed")
	})

	t.Run("Find", func(t *testing.T) {
		findTX, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = findTX.Rollback()
		})
		createParams2 := createParams
		task1, err := models.CreateScheduledTask(findTX.Querier, createParams2)
		require.NoError(t, err)

		createParams2.Disabled = true
		task2, err := models.CreateScheduledTask(findTX.Querier, createParams2)
		require.NoError(t, err)

		createParams2.Disabled = false
		createParams2.Type = models.ScheduledMySQLBackupTask
		createParams2.Data = models.ScheduledTaskData{
			MySQLBackupTask: &models.MySQLBackupTaskData{
				ServiceID:  "svc1",
				LocationID: "loc1",
				Name:       "mysql",
			},
		}
		task3, err := models.CreateScheduledTask(findTX.Querier, createParams2)
		require.NoError(t, err)

		createParams2.Type = models.ScheduledMongoDBBackupTask
		createParams2.Data = models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
				ServiceID:  "svc2",
				LocationID: "loc1",
				Name:       "mongo",
			},
		}
		task4, err := models.CreateScheduledTask(findTX.Querier, createParams2)
		require.NoError(t, err)
		type testCase struct {
			filter models.ScheduledTasksFilter
			ids    []string
		}

		tests := []testCase{
			{
				filter: models.ScheduledTasksFilter{},
				ids:    []string{task1.ID, task2.ID, task3.ID, task4.ID},
			},
			{
				filter: models.ScheduledTasksFilter{
					Disabled: pointer.ToBool(true),
				},
				ids: []string{task2.ID},
			},
			{
				filter: models.ScheduledTasksFilter{
					Disabled: pointer.ToBool(false),
				},
				ids: []string{task1.ID, task3.ID, task4.ID},
			},
			{
				filter: models.ScheduledTasksFilter{
					Types: []models.ScheduledTaskType{
						models.ScheduledMongoDBBackupTask,
					},
				},
				ids: []string{task4.ID},
			},
			{
				filter: models.ScheduledTasksFilter{
					LocationID: "loc1",
				},
				ids: []string{task3.ID, task4.ID},
			},
			{
				filter: models.ScheduledTasksFilter{
					ServiceID: "svc2",
				},
				ids: []string{task4.ID},
			},
		}

		for _, tc := range tests {
			tasks, err := models.FindScheduledTasks(findTX.Querier, tc.filter)
			assert.NoError(t, err)
			ids := make([]string, 0, len(tasks))
			for _, task := range tasks {
				ids = append(ids, task.ID)
			}
			sort.Strings(tc.ids)
			sort.Strings(ids)
			assert.Equal(t, tc.ids, ids)
		}
	})
}
