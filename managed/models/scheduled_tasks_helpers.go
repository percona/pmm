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

package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// FindScheduledTaskByID finds ScheduledTask by ID.
func FindScheduledTaskByID(q *reform.Querier, id string) (*ScheduledTask, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty ScheduledTask ID.")
	}

	res := &ScheduledTask{ID: id}
	err := q.Reload(res)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, errors.Wrapf(ErrNotFound, "couldn't get scheduled task with ID %q", id)
		}
		return nil, errors.WithStack(err)
	}

	return res, nil
}

// ScheduledTasksFilter represents filters for scheduled tasks.
type ScheduledTasksFilter struct {
	Disabled    *bool
	Types       []ScheduledTaskType
	ServiceID   string
	ClusterName string
	LocationID  string
	Mode        BackupMode
	Compression BackupCompression
	Name        string
	Folder      *string
}

// FindScheduledTasks returns all scheduled tasks satisfying filter.
func FindScheduledTasks(q *reform.Querier, filters ScheduledTasksFilter) ([]*ScheduledTask, error) {
	var args []interface{}
	var andConds []string
	idx := 1
	if len(filters.Types) != 0 {
		p := strings.Join(q.Placeholders(idx, len(filters.Types)), ", ")
		for _, fType := range filters.Types {
			args = append(args, fType)
		}
		idx += len(filters.Types)
		andConds = append(andConds, fmt.Sprintf("type IN (%s)", p))
	}

	if filters.Disabled != nil {
		cond := "disabled IS "
		if *filters.Disabled {
			cond += "TRUE"
		} else {
			cond += "FALSE"
		}
		andConds = append(andConds, cond)
	}

	crossJoin := false
	if filters.ServiceID != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'service_id' = "+q.Placeholder(idx))
		args = append(args, filters.ServiceID)
		idx++
	}
	if filters.ClusterName != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'cluster_name' = "+q.Placeholder(idx))
		args = append(args, filters.ClusterName)
		idx++
	}
	if filters.LocationID != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'location_id' = "+q.Placeholder(idx))
		args = append(args, filters.LocationID)
		idx++
	}
	if filters.Mode != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'mode' = "+q.Placeholder(idx))
		args = append(args, filters.Mode)
		idx++
	}
	if filters.Compression != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'compression' = "+q.Placeholder(idx))
		args = append(args, filters.Compression)
		idx++
	}
	if filters.Name != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'name' = "+q.Placeholder(idx))
		args = append(args, filters.Name)
		idx++
	}
	if filters.Folder != nil {
		crossJoin = true
		andConds = append(andConds, "value ->> 'folder' = "+q.Placeholder(idx))
		args = append(args, *filters.Folder)
		// idx++
	}

	var tail strings.Builder
	if crossJoin {
		tail.WriteString("CROSS JOIN jsonb_each(data) ")
	}

	if len(andConds) != 0 {
		tail.WriteString("WHERE ")
		tail.WriteString(strings.Join(andConds, " AND "))
		tail.WriteRune(' ')
	}
	tail.WriteString("ORDER BY created_at DESC")

	structs, err := q.SelectAllFrom(ScheduledTaskTable, tail.String(), args...)
	if err != nil {
		return nil, err
	}
	tasks := make([]*ScheduledTask, len(structs))
	for i, s := range structs {
		tasks[i] = s.(*ScheduledTask) //nolint:forcetypeassert
	}
	return tasks, nil
}

// CreateScheduledTaskParams are params for creating new scheduled task.
type CreateScheduledTaskParams struct {
	CronExpression string
	StartAt        time.Time
	NextRun        time.Time
	Type           ScheduledTaskType
	Data           *ScheduledTaskData
	Disabled       bool
}

// Validate checks if required params are set and valid.
func (p CreateScheduledTaskParams) Validate() error {
	switch p.Type {
	case ScheduledMySQLBackupTask:
	case ScheduledMongoDBBackupTask:
	default:
		return status.Errorf(codes.InvalidArgument, "Unknown type: %s", p.Type)
	}
	_, err := cron.ParseStandard(p.CronExpression)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Invalid cron expression: %v", err)
	}

	return nil
}

// CreateScheduledTask creates scheduled task.
// Must be performed in transaction.
func CreateScheduledTask(q *reform.Querier, params CreateScheduledTaskParams) (*ScheduledTask, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	newName, err := nameFromTaskData(params.Type, params.Data)
	if err != nil {
		return nil, err
	}

	if err := checkUniqueScheduledTaskName(q, newName); err != nil {
		return nil, errors.Wrapf(err, "couldn't create task with name %s", newName)
	}

	id := "/scheduled_task_id/" + uuid.New().String()
	if err := checkUniqueScheduledTaskID(q, id); err != nil {
		return nil, err
	}

	task := &ScheduledTask{
		ID:             id,
		CronExpression: params.CronExpression,
		Disabled:       params.Disabled,
		StartAt:        params.StartAt,
		NextRun:        params.NextRun,
		Type:           params.Type,
		Data:           params.Data,
	}
	if err := q.Insert(task); err != nil {
		return nil, errors.WithStack(err)
	}
	return task, nil
}

// ChangeScheduledTaskParams are params for updating existing schedule task.
type ChangeScheduledTaskParams struct {
	NextRun        *time.Time
	LastRun        *time.Time
	Disable        *bool
	Running        *bool
	Error          *string
	Data           *ScheduledTaskData
	CronExpression *string
}

// Validate checks if params for scheduled tasks are valid.
func (p *ChangeScheduledTaskParams) Validate() error {
	if p.CronExpression != nil {
		_, err := cron.ParseStandard(*p.CronExpression)
		if err != nil {
			return err
		}
	}
	return nil
}

// ChangeScheduledTask updates existing scheduled task.
// Must be performed in transaction.
func ChangeScheduledTask(q *reform.Querier, id string, params ChangeScheduledTaskParams) (*ScheduledTask, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	row, err := FindScheduledTaskByID(q, id)
	if err != nil {
		return nil, err
	}

	if params.NextRun != nil {
		row.NextRun = *params.NextRun
	}

	if params.LastRun != nil {
		row.LastRun = *params.LastRun
	}

	if params.Disable != nil {
		row.Disabled = *params.Disable
	}

	if params.Running != nil {
		row.Running = *params.Running
	}

	if params.Data != nil {
		newName, err := nameFromTaskData(row.Type, params.Data)
		if err != nil {
			return nil, err
		}
		oldName, err := nameFromTaskData(row.Type, row.Data)
		if err != nil {
			return nil, err
		}

		if newName != oldName {
			if err := checkUniqueScheduledTaskName(q, newName); err != nil {
				return nil, errors.Wrapf(err, "couldn't change task name to %s", newName)
			}
		}

		row.Data = params.Data
	}

	if params.CronExpression != nil {
		row.CronExpression = *params.CronExpression
	}

	if params.Error != nil {
		row.Error = *params.Error
	}

	if err := q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update scheduled task")
	}

	return row, nil
}

// RemoveScheduledTask removes task from DB.
func RemoveScheduledTask(q *reform.Querier, id string) error {
	if _, err := FindScheduledTaskByID(q, id); err != nil {
		return err
	}
	if err := q.Delete(&ScheduledTask{ID: id}); err != nil {
		return errors.Wrap(err, "failed to delete scheduled task")
	}

	return nil
}

func checkUniqueScheduledTaskID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty schedule task ID")
	}

	task := &ScheduledTask{ID: id}
	err := q.Reload(task)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Scheduled task with ID %q already exists.", id)
}

func checkUniqueScheduledTaskName(q *reform.Querier, name string) error {
	tasks, err := FindScheduledTasks(q, ScheduledTasksFilter{Name: name})
	if err != nil {
		return err
	}
	if len(tasks) != 0 {
		return ErrAlreadyExists
	}
	return nil
}

func nameFromTaskData(taskType ScheduledTaskType, taskData *ScheduledTaskData) (string, error) {
	if taskData != nil {
		switch taskType {
		case ScheduledMySQLBackupTask:
			if taskData.MySQLBackupTask != nil {
				return taskData.MySQLBackupTask.Name, nil
			}
		case ScheduledMongoDBBackupTask:
			if taskData.MongoDBBackupTask != nil {
				return taskData.MongoDBBackupTask.Name, nil
			}
		default:
			return "", status.Errorf(codes.InvalidArgument, "Unknown type: %s", taskType)
		}
	}
	return "", errors.New("scheduled task name cannot be empty")
}

// Retention returns how many backup artifacts should be stored for the task.
func (s *ScheduledTask) Retention() (uint32, error) {
	data, err := s.CommonBackupData()
	if err != nil {
		return 0, err
	}
	return data.Retention, nil
}

// Mode returns task backup mode.
func (s *ScheduledTask) Mode() (BackupMode, error) {
	data, err := s.CommonBackupData()
	if err != nil {
		return "", err
	}
	return data.Mode, nil
}

// Compression returns backup compression.
func (s *ScheduledTask) Compression() (BackupCompression, error) {
	data, err := s.CommonBackupData()
	if err != nil {
		return "", err
	}
	return data.Compression, nil
}

// LocationID returns task location.
func (s *ScheduledTask) LocationID() (string, error) {
	data, err := s.CommonBackupData()
	if err != nil {
		return "", err
	}
	return data.LocationID, nil
}

// ServiceID returns task service ID.
func (s *ScheduledTask) ServiceID() (string, error) {
	data, err := s.CommonBackupData()
	if err != nil {
		return "", err
	}
	return data.ServiceID, nil
}

// CommonBackupData returns the common backup data for the scheduled task.
func (s *ScheduledTask) CommonBackupData() (*CommonBackupTaskData, error) {
	if s.Data != nil {
		switch s.Type {
		case ScheduledMySQLBackupTask:
			if s.Data.MySQLBackupTask != nil {
				return &s.Data.MySQLBackupTask.CommonBackupTaskData, nil
			}
		case ScheduledMongoDBBackupTask:
			if s.Data.MongoDBBackupTask != nil {
				return &s.Data.MongoDBBackupTask.CommonBackupTaskData, nil
			}
		default:
			return nil, errors.Errorf("invalid backup type %s of scheduled task %s", s.Type, s.ID)
		}
	}

	return nil, errors.Errorf("empty backup data of scheduled task %s", s.ID)
}
