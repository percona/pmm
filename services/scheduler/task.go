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
	"time"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/backup"
)

// Task represents task which will be run inside scheduler.
type Task interface {
	Run(ctx context.Context) error
	Type() models.ScheduledTaskType
	Data() models.ScheduledTaskData
	ID() string
	SetID(string)
}

// common implementation for all tasks.
type common struct {
	id string
}

func (c *common) ID() string {
	return c.id
}

func (c *common) SetID(id string) {
	c.id = id
}

// CommonBackupTaskParams contains common fields for all backup tasks.
type CommonBackupTaskParams struct {
	ServiceID     string
	LocationID    string
	Name          string
	Description   string
	Retention     uint32
	Retries       uint32
	RetryInterval time.Duration
}

type mySQLBackupTask struct {
	*common
	backupService backupService
	CommonBackupTaskParams
}

// NewMySQLBackupTask create new task for mysql backup.
func NewMySQLBackupTask(backupService backupService, params CommonBackupTaskParams) Task {
	return &mySQLBackupTask{
		common:                 &common{},
		backupService:          backupService,
		CommonBackupTaskParams: params,
	}
}

func (t *mySQLBackupTask) Run(ctx context.Context) error {
	name := t.Name + "_" + time.Now().Format(time.RFC3339)
	_, err := t.backupService.PerformBackup(ctx, backup.PerformBackupParams{
		ServiceID:  t.ServiceID,
		LocationID: t.LocationID,
		Name:       name,
		ScheduleID: t.ID(),
	})
	return err
}

func (t *mySQLBackupTask) Type() models.ScheduledTaskType {
	return models.ScheduledMySQLBackupTask
}

func (t *mySQLBackupTask) Data() models.ScheduledTaskData {
	return models.ScheduledTaskData{
		MySQLBackupTask: &models.MySQLBackupTaskData{
			CommonBackupTaskData: models.CommonBackupTaskData{
				ServiceID:     t.ServiceID,
				LocationID:    t.LocationID,
				Name:          t.Name,
				Description:   t.Description,
				Retention:     t.Retention,
				Retries:       t.Retries,
				RetryInterval: t.RetryInterval,
			},
		},
	}
}

type mongoBackupTask struct {
	*common
	backupService backupService
	CommonBackupTaskParams
}

// NewMongoBackupTask create new task for mongo backup.
func NewMongoBackupTask(backupService backupService, params CommonBackupTaskParams) Task {
	return &mongoBackupTask{
		common:                 &common{},
		backupService:          backupService,
		CommonBackupTaskParams: params,
	}
}

func (t *mongoBackupTask) Run(ctx context.Context) error {
	name := t.Name + "_" + time.Now().Format(time.RFC3339)
	_, err := t.backupService.PerformBackup(ctx, backup.PerformBackupParams{
		ServiceID:  t.ServiceID,
		LocationID: t.LocationID,
		Name:       name,
		ScheduleID: t.ID(),
	})
	return err
}

func (t *mongoBackupTask) Type() models.ScheduledTaskType {
	return models.ScheduledMongoDBBackupTask
}

func (t *mongoBackupTask) Data() models.ScheduledTaskData {
	return models.ScheduledTaskData{
		MongoDBBackupTask: &models.MongoBackupTaskData{
			CommonBackupTaskData: models.CommonBackupTaskData{
				ServiceID:     t.ServiceID,
				LocationID:    t.LocationID,
				Name:          t.Name,
				Description:   t.Description,
				Retention:     t.Retention,
				Retries:       t.Retries,
				RetryInterval: t.RetryInterval,
			},
		},
	}
}
