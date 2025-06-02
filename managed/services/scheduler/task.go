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
	"time"

	"github.com/pkg/errors"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/backup"
)

// Task represents task which will be run inside scheduler.
type Task interface {
	Run(ctx context.Context, scheduler *Service) error
	ID() string
	Type() models.ScheduledTaskType
	Data() *models.ScheduledTaskData
}

// common implementation for all tasks.
type common struct {
	id string
}

func (c *common) ID() string {
	return c.id
}

// BackupTaskParams contains common fields for all backup tasks.
type BackupTaskParams struct {
	ServiceID     string
	ClusterName   string
	LocationID    string
	Name          string
	Description   string
	DataModel     models.DataModel
	Mode          models.BackupMode
	Retention     uint32
	Retries       uint32
	RetryInterval time.Duration
	Folder        string
}

// Validate checks backup task parameters for correctness.
func (p *BackupTaskParams) Validate() error {
	if p.Name == "" {
		return errors.New("backup name can't be empty")
	}

	if p.ServiceID == "" {
		return errors.New("service id can't be empty")
	}

	if p.LocationID == "" {
		return errors.New("location id can't be empty")
	}

	if err := p.DataModel.Validate(); err != nil {
		return err
	}

	return p.Mode.Validate()
}

type mySQLBackupTask struct {
	common
	*BackupTaskParams
}

// NewMySQLBackupTask create new task for mysql backup.
func NewMySQLBackupTask(params *BackupTaskParams) (Task, error) { //nolint:ireturn,nolintlint
	if err := params.Validate(); err != nil {
		return nil, err
	}

	if params.Mode != models.Snapshot {
		return nil, errors.Errorf("unsupported backup mode for mySQL: %s", params.Mode)
	}

	if params.DataModel != models.PhysicalDataModel {
		return nil, errors.Errorf("unsupported backup data model for mySQL: %s", params.DataModel)
	}

	return &mySQLBackupTask{
		BackupTaskParams: params,
	}, nil
}

func (t *mySQLBackupTask) Run(ctx context.Context, scheduler *Service) error {
	_, err := scheduler.backupService.PerformBackup(ctx, backup.PerformBackupParams{
		ServiceID:     t.ServiceID,
		LocationID:    t.LocationID,
		Name:          t.Name,
		Mode:          t.Mode,
		DataModel:     t.DataModel,
		ScheduleID:    t.ID(),
		Retries:       t.Retries,
		RetryInterval: t.RetryInterval,
		Folder:        t.Folder,
	})
	return err
}

func (t *mySQLBackupTask) Type() models.ScheduledTaskType {
	return models.ScheduledMySQLBackupTask
}

func (t *mySQLBackupTask) Data() *models.ScheduledTaskData {
	return &models.ScheduledTaskData{
		MySQLBackupTask: &models.MySQLBackupTaskData{
			CommonBackupTaskData: models.CommonBackupTaskData{
				ServiceID:     t.ServiceID,
				ClusterName:   t.ClusterName,
				LocationID:    t.LocationID,
				Name:          t.Name,
				Description:   t.Description,
				Retention:     t.Retention,
				DataModel:     t.DataModel,
				Mode:          t.Mode,
				Retries:       t.Retries,
				RetryInterval: t.RetryInterval,
				Folder:        t.Folder,
			},
		},
	}
}

type mongoDBBackupTask struct {
	common
	*BackupTaskParams
}

// NewMongoDBBackupTask create new task for mongo backup.
func NewMongoDBBackupTask(params *BackupTaskParams) (Task, error) { //nolint:ireturn,nolintlint
	if err := params.Validate(); err != nil {
		return nil, err
	}

	if params.Mode != models.Snapshot && params.Mode != models.PITR {
		return nil, errors.Errorf("unsupported backup mode for mongoDB: %s", params.Mode)
	}

	if params.Mode == models.PITR && params.DataModel != models.LogicalDataModel {
		return nil, errors.WithMessage(backup.ErrIncompatibleDataModel, "PITR is only supported for logical backups")
	}

	return &mongoDBBackupTask{
		BackupTaskParams: params,
	}, nil
}

func (t *mongoDBBackupTask) Run(ctx context.Context, scheduler *Service) error {
	_, err := scheduler.backupService.PerformBackup(ctx, backup.PerformBackupParams{
		ServiceID:     t.ServiceID,
		LocationID:    t.LocationID,
		Name:          t.Name,
		DataModel:     t.DataModel,
		Mode:          t.Mode,
		ScheduleID:    t.ID(),
		Retries:       t.Retries,
		RetryInterval: t.RetryInterval,
		Folder:        t.Folder,
	})
	return err
}

func (t *mongoDBBackupTask) Type() models.ScheduledTaskType {
	return models.ScheduledMongoDBBackupTask
}

func (t *mongoDBBackupTask) Data() *models.ScheduledTaskData {
	return &models.ScheduledTaskData{
		MongoDBBackupTask: &models.MongoBackupTaskData{
			CommonBackupTaskData: models.CommonBackupTaskData{
				ServiceID:     t.ServiceID,
				ClusterName:   t.ClusterName,
				LocationID:    t.LocationID,
				Name:          t.Name,
				Description:   t.Description,
				DataModel:     t.DataModel,
				Mode:          t.Mode,
				Retention:     t.Retention,
				Retries:       t.Retries,
				RetryInterval: t.RetryInterval,
				Folder:        t.Folder,
			},
		},
	}
}
