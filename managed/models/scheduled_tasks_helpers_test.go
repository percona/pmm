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

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestScheduledTaskHelpers(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	createParams1 := models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMySQLBackupTask,
		Data: &models.ScheduledTaskData{
			MySQLBackupTask: &models.MySQLBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "",
					LocationID:  "",
					Name:        "task1",
					Description: "",
				},
			},
		},
		Disabled: false,
	}

	createParams1DuplicateName := models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMySQLBackupTask,
		Data: &models.ScheduledTaskData{
			MySQLBackupTask: &models.MySQLBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "",
					LocationID:  "",
					Name:        "task1",
					Description: "",
				},
			},
		},
		Disabled: false,
	}

	createParams2 := models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMySQLBackupTask,
		Data: &models.ScheduledTaskData{
			MySQLBackupTask: &models.MySQLBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "",
					LocationID:  "",
					Name:        "task2",
					Description: "",
				},
			},
		},
		Disabled: true,
	}

	createParams3 := models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMySQLBackupTask,
		Data: &models.ScheduledTaskData{
			MySQLBackupTask: &models.MySQLBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "svc1",
					LocationID:  "loc1",
					Name:        "mysql",
					Description: "",
				},
			},
		},
		Disabled: false,
	}

	createParams4 := models.CreateScheduledTaskParams{
		CronExpression: "* * * * *",
		Type:           models.ScheduledMongoDBBackupTask,
		Data: &models.ScheduledTaskData{
			MongoDBBackupTask: &models.MongoBackupTaskData{
				CommonBackupTaskData: models.CommonBackupTaskData{
					ServiceID:   "svc2",
					ClusterName: "cluster",
					LocationID:  "loc1",
					Name:        "mongo",
				},
			},
		},
		Disabled: false,
	}

	t.Run("CreateAndFind", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		task, err := models.CreateScheduledTask(tx.Querier, createParams1)
		require.NoError(t, err)

		task, err = models.FindScheduledTaskByID(tx.Querier, task.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, task.ID)
		assert.Equal(t, createParams1.CronExpression, task.CronExpression)
		assert.Equal(t, createParams1.Type, task.Type)
		assert.Equal(t, createParams1.Disabled, task.Disabled)
		require.NotNil(t, task.Data.MySQLBackupTask)
		assert.Equal(t, createParams1.Data.MySQLBackupTask.Name, task.Data.MySQLBackupTask.Name)

		_, err = models.CreateScheduledTask(tx.Querier, models.CreateScheduledTaskParams{
			CronExpression: "a * * * *",
			Type:           models.ScheduledMySQLBackupTask,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid cron expression")

		// Cannot create with the existing name.
		_, err = models.CreateScheduledTask(tx.Querier, createParams1DuplicateName)
		require.ErrorIs(t, err, models.ErrAlreadyExists)

		tasks, err := models.FindScheduledTasks(tx.Querier, models.ScheduledTasksFilter{Name: createParams1.Data.MySQLBackupTask.Name})
		require.NoError(t, err)
		assert.Equal(t, []*models.ScheduledTask{task}, tasks)
	})

	t.Run("Change", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		task1, err := models.CreateScheduledTask(tx.Querier, createParams1)
		require.NoError(t, err)

		changeParams1 := models.ChangeScheduledTaskParams{
			NextRun: pointer.ToTime(time.Now()),
			LastRun: pointer.ToTime(time.Now()),
			Disable: pointer.ToBool(true),
			Running: pointer.ToBool(true),
			Error:   pointer.ToString("something"),
		}
		task1, err = models.ChangeScheduledTask(tx.Querier, task1.ID, changeParams1)
		assert.NoError(t, err)
		assert.Equal(t, *changeParams1.NextRun, task1.NextRun)
		assert.Equal(t, *changeParams1.LastRun, task1.LastRun)
		assert.Equal(t, *changeParams1.Disable, task1.Disabled)
		assert.Equal(t, *changeParams1.Running, task1.Running)
		assert.Equal(t, *changeParams1.Error, task1.Error)

		// Cannot change to the existing name.
		task2, err := models.CreateScheduledTask(tx.Querier, createParams2)
		require.NoError(t, err)
		changeParams2 := models.ChangeScheduledTaskParams{
			Data: &models.ScheduledTaskData{
				MySQLBackupTask: &models.MySQLBackupTaskData{
					CommonBackupTaskData: models.CommonBackupTaskData{
						Name: task1.Data.MySQLBackupTask.Name,
					},
				},
			},
		}
		_, err = models.ChangeScheduledTask(tx.Querier, task2.ID, changeParams2)
		assert.ErrorIs(t, err, models.ErrAlreadyExists)
	})

	t.Run("Remove", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		task, err := models.CreateScheduledTask(tx.Querier, createParams1)
		require.NoError(t, err)

		err = models.RemoveScheduledTask(tx.Querier, task.ID)
		assert.NoError(t, err)

		_, err = models.FindScheduledTaskByID(tx.Querier, task.ID)
		assert.ErrorIs(t, err, models.ErrNotFound, "task is not removed")
	})

	t.Run("Find", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		task1, err := models.CreateScheduledTask(tx.Querier, createParams1)
		require.NoError(t, err)

		task2, err := models.CreateScheduledTask(tx.Querier, createParams2)
		require.NoError(t, err)

		task3, err := models.CreateScheduledTask(tx.Querier, createParams3)
		require.NoError(t, err)

		task4, err := models.CreateScheduledTask(tx.Querier, createParams4)
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
			{
				filter: models.ScheduledTasksFilter{
					ClusterName: "cluster",
				},
				ids: []string{task4.ID},
			},
		}

		for _, tc := range tests {
			tasks, err := models.FindScheduledTasks(tx.Querier, tc.filter)
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
