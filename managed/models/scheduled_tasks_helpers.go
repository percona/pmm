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
	switch err := q.Reload(res); err {
	case nil:
		return res, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "ScheduledTask with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// ScheduledTasksFilter represents filters for scheduled tasks.
type ScheduledTasksFilter struct {
	Disabled   *bool
	Types      []ScheduledTaskType
	ServiceID  string
	LocationID string
	Mode       BackupMode
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
		tasks[i] = s.(*ScheduledTask)
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
func CreateScheduledTask(q *reform.Querier, params CreateScheduledTaskParams) (*ScheduledTask, error) {
	if err := params.Validate(); err != nil {
		return nil, err
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
func (p ChangeScheduledTaskParams) Validate() error {
	if p.CronExpression != nil {
		_, err := cron.ParseStandard(*p.CronExpression)
		if err != nil {
			return err
		}
	}
	return nil
}

// ChangeScheduledTask updates existing scheduled task.
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
	switch err := q.Reload(task); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Scheduled task with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}
