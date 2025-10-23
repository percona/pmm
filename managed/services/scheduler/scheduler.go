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

// Package scheduler implements scheduler.
package scheduler

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/go-co-op/gocron"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// Service is responsible for executing tasks and storing them to DB.
type Service struct {
	db            *reform.DB
	l             *logrus.Entry
	backupService backupService

	mx        sync.Mutex
	scheduler *gocron.Scheduler

	taskMx sync.RWMutex
	tasks  map[string]context.CancelFunc

	jobsMx sync.RWMutex
	jobs   map[string]*gocron.Job
}

// New creates new scheduler service.
func New(db *reform.DB, backupService backupService) *Service {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.TagsUnique()
	scheduler.WaitForScheduleAll()
	return &Service{
		db:            db,
		scheduler:     scheduler,
		l:             logrus.WithField("component", "scheduler"),
		backupService: backupService,
		tasks:         make(map[string]context.CancelFunc),
		jobs:          make(map[string]*gocron.Job),
	}
}

// Run loads tasks from DB and starts scheduler.
func (s *Service) Run(ctx context.Context) {
	if err := s.loadFromDB(); err != nil { //nolint:contextcheck
		s.l.Warn(err)
	}
	s.scheduler.StartAsync()
	<-ctx.Done()
	s.scheduler.Stop()
}

// AddParams contains parameters for adding new add to service.
type AddParams struct {
	CronExpression string
	Disabled       bool
	StartAt        time.Time
}

// Add adds task to scheduler and save it to DB.
func (s *Service) Add(task Task, params AddParams) (*models.ScheduledTask, error) {
	var scheduledTask *models.ScheduledTask

	// This transaction is valid only with serializable isolation level. On lower isolation levels it can produce anomalies.
	errTx := s.db.InTransactionContext(s.db.Querier.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
		var err error
		if err = checkAddPreconditions(tx.Querier, task.Data(), !params.Disabled, ""); err != nil {
			return err
		}
		scheduledTask, err = models.CreateScheduledTask(tx.Querier, models.CreateScheduledTaskParams{
			CronExpression: params.CronExpression,
			StartAt:        params.StartAt,
			Type:           task.Type(),
			Data:           task.Data(),
			Disabled:       params.Disabled,
		})
		if err != nil {
			return err
		}

		if err = s.addDBTask(scheduledTask); err != nil {
			return err
		}

		s.jobsMx.RLock()
		scheduleJob := s.jobs[scheduledTask.ID]
		s.jobsMx.RUnlock()

		// If it's not disabled, update next run.
		if scheduleJob != nil {
			scheduledTask, err = models.ChangeScheduledTask(tx.Querier, scheduledTask.ID, models.ChangeScheduledTaskParams{
				NextRun: pointer.ToTime(scheduleJob.NextRun().UTC()),
				LastRun: pointer.ToTime(scheduleJob.LastRun().UTC()),
			})
			if err != nil {
				s.l.WithField("id", scheduledTask.ID).Errorf("failed to set next run for new created task")
				s.mx.Lock()
				s.scheduler.RemoveByReference(scheduleJob)
				s.mx.Unlock()
				return err
			}
		}

		return nil
	})
	return scheduledTask, errTx
}

// Remove stops task specified by id and removes it from DB and scheduler.
func (s *Service) Remove(id string) error {
	s.taskMx.RLock()
	if cancel, ok := s.tasks[id]; ok {
		cancel()
	}
	s.taskMx.RUnlock()

	s.jobsMx.Lock()
	delete(s.jobs, id)
	s.jobsMx.Unlock()

	err := s.db.InTransaction(func(tx *reform.TX) error {
		return models.RemoveScheduledTask(tx.Querier, id)
	})
	if err != nil {
		return err
	}

	s.mx.Lock()
	_ = s.scheduler.RemoveByTag(id)
	s.mx.Unlock()

	return nil
}

// Update changes scheduled task in DB and re-add it to scheduler.
func (s *Service) Update(id string, params models.ChangeScheduledTaskParams) error {
	return s.db.InTransactionContext(s.db.Querier.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
		if err := checkUpdatePreconditions(tx.Querier, params.Data, !pointer.GetBool(params.Disable), id); err != nil {
			return err
		}

		scheduledTask, err := models.ChangeScheduledTask(tx.Querier, id, params)
		if err != nil {
			return err
		}

		s.mx.Lock()
		// TODO if addDBTask will fail, then scheduler state will be not restored by the transaction rollback
		_ = s.scheduler.RemoveByTag(id)
		s.mx.Unlock()

		return s.addDBTask(scheduledTask)
	})
}

func (s *Service) loadFromDB() error {
	dbTasks, err := models.FindScheduledTasks(s.db.Querier, models.ScheduledTasksFilter{
		Disabled: pointer.ToBool(false),
	})
	if err != nil {
		return err
	}

	s.mx.Lock()
	s.scheduler.Clear()
	s.mx.Unlock()

	for _, dbTask := range dbTasks {
		if err := s.addDBTask(dbTask); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) addDBTask(dbTask *models.ScheduledTask) error {
	if dbTask.Disabled {
		return nil
	}

	task, err := s.convertDBTask(dbTask)
	if err != nil {
		return err
	}

	s.mx.Lock()
	fn := s.wrapTask(task, dbTask.ID)
	j := s.scheduler.Cron(dbTask.CronExpression).SingletonMode()
	if !dbTask.StartAt.IsZero() {
		j = j.StartAt(dbTask.StartAt)
	}
	scheduleJob, err := j.Tag(dbTask.ID).Do(fn)
	if err != nil {
		s.mx.Unlock()
		return err
	}
	s.mx.Unlock()

	s.jobsMx.Lock()
	s.jobs[dbTask.ID] = scheduleJob
	s.jobsMx.Unlock()
	return nil
}

func (s *Service) wrapTask(task Task, id string) func() {
	return func() {
		var err error
		l := s.l.WithFields(logrus.Fields{
			"id":       id,
			"taskType": task.Type(),
		})
		ctx, cancel := context.WithCancel(context.Background())

		s.taskMx.Lock()
		s.tasks[id] = cancel
		s.taskMx.Unlock()

		defer func() {
			cancel()
			s.taskMx.Lock()
			delete(s.tasks, id)
			s.taskMx.Unlock()
		}()

		t := time.Now()
		l.Debug("Starting task")
		_, err = models.ChangeScheduledTask(s.db.Querier, id, models.ChangeScheduledTaskParams{
			Running: pointer.ToBool(true),
		})
		if err != nil {
			l.Errorf("failed to change running state: %v", err)
		}

		taskErr := task.Run(ctx, s)
		if taskErr != nil {
			l.Error(taskErr)
		}
		l.WithField("duration", time.Since(t)).Debug("Ended task")

		s.taskFinished(id, taskErr)
	}
}

func (s *Service) taskFinished(id string, taskErr error) {
	s.jobsMx.RLock()
	job := s.jobs[id]
	s.jobsMx.RUnlock()

	l := s.l.WithField("id", id)

	txErr := s.db.InTransaction(func(tx *reform.TX) error {
		params := models.ChangeScheduledTaskParams{
			Running: pointer.ToBool(false),
		}

		if taskErr != nil {
			params.Error = pointer.ToString(taskErr.Error())
		} else {
			params.Error = pointer.ToString("")
		}

		if job != nil {
			params.NextRun = pointer.ToTime(job.NextRun().UTC())
			params.LastRun = pointer.ToTime(job.LastRun().UTC())
		} else {
			l.Errorf("failed to find scheduled task")
		}

		_, err := models.ChangeScheduledTask(tx.Querier, id, params)
		if err != nil {
			return err
		}
		return nil
	})

	if txErr != nil {
		l.Errorf("failed to commit finished task: %v", txErr)
	}
}

func (s *Service) convertDBTask(dbTask *models.ScheduledTask) (Task, error) { //nolint:ireturn
	var task Task
	switch dbTask.Type {
	case models.ScheduledMySQLBackupTask:
		data := dbTask.Data.MySQLBackupTask
		task = &mySQLBackupTask{
			common: common{
				id: dbTask.ID,
			},
			BackupTaskParams: &BackupTaskParams{
				ServiceID:     data.ServiceID,
				LocationID:    data.LocationID,
				Name:          data.Name,
				Description:   data.Description,
				DataModel:     data.DataModel,
				Mode:          data.Mode,
				Retention:     data.Retention,
				Retries:       data.Retries,
				RetryInterval: data.RetryInterval,
				Folder:        data.Folder,
				Compression:   data.Compression,
			},
		}
	case models.ScheduledMongoDBBackupTask:
		data := dbTask.Data.MongoDBBackupTask
		task = &mongoDBBackupTask{
			common: common{
				id: dbTask.ID,
			},
			BackupTaskParams: &BackupTaskParams{
				ServiceID:     data.ServiceID,
				LocationID:    data.LocationID,
				Name:          data.Name,
				Description:   data.Description,
				DataModel:     data.DataModel,
				Mode:          data.Mode,
				Retention:     data.Retention,
				Retries:       data.Retries,
				RetryInterval: data.RetryInterval,
				Folder:        data.Folder,
				Compression:   data.Compression,
			},
		}

	default:
		return nil, errors.Errorf("unknown task type: %s", dbTask.Type)
	}

	return task, nil
}

func checkAddPreconditions(q *reform.Querier, data *models.ScheduledTaskData, enabled bool, scheduledTaskID string) error {
	switch {
	case data.MySQLBackupTask != nil:
		if err := services.CheckArtifactOverlapping(q, data.MySQLBackupTask.ServiceID, data.MySQLBackupTask.LocationID, data.MySQLBackupTask.Folder); err != nil {
			return err
		}
	case data.MongoDBBackupTask != nil:
		if err := services.CheckArtifactOverlapping(q, data.MongoDBBackupTask.ServiceID, data.MongoDBBackupTask.LocationID, data.MongoDBBackupTask.Folder); err != nil {
			return err
		}
		if enabled {
			return services.CheckMongoDBBackupPreconditions(
				q,
				data.MongoDBBackupTask.Mode,
				data.MongoDBBackupTask.ClusterName,
				data.MongoDBBackupTask.ServiceID,
				scheduledTaskID)
		}
	}
	return nil
}

func checkUpdatePreconditions(q *reform.Querier, data *models.ScheduledTaskData, enabled bool, scheduledTaskID string) error {
	switch {
	case data.MySQLBackupTask != nil:
	case data.MongoDBBackupTask != nil:
		if enabled {
			return services.CheckMongoDBBackupPreconditions(
				q,
				data.MongoDBBackupTask.Mode,
				data.MongoDBBackupTask.ClusterName,
				data.MongoDBBackupTask.ServiceID,
				scheduledTaskID)
		}
	}
	return nil
}
