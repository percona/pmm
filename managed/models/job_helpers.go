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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

const (
	defaultLimit = 50
)

// FindJobByID finds Job by ID.
func FindJobByID(q *reform.Querier, id string) (*Job, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Job ID.")
	}

	res := &Job{ID: id}

	err := q.Reload(res)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Job with ID %q not found.", id)
		}
		return nil, errors.WithStack(err)
	}

	return res, nil
}

// JobsFilter represents filter for jobs.
type JobsFilter struct {
	ArtifactID string
	RestoreID  string
	Types      []JobType
}

// FindJobs returns logs satisfying filters.
func FindJobs(q *reform.Querier, filters JobsFilter) ([]*Job, error) {
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

	crossJoin := false
	if filters.ArtifactID != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'artifact_id' = "+q.Placeholder(idx))
		args = append(args, filters.ArtifactID)
		idx++
	}

	if filters.RestoreID != "" {
		crossJoin = true
		andConds = append(andConds, "value ->> 'restore_id' = "+q.Placeholder(idx))
		args = append(args, filters.RestoreID)
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

	structs, err := q.SelectAllFrom(JobTable, tail.String(), args...)
	if err != nil {
		return nil, err
	}
	jobs := make([]*Job, len(structs))
	for i, s := range structs {
		jobs[i] = s.(*Job) //nolint:forcetypeassert
	}
	return jobs, nil
}

// CreateJobParams are params for creating a new job.
type CreateJobParams struct {
	PMMAgentID string
	Type       JobType
	Data       *JobData
	Timeout    time.Duration
	Interval   time.Duration
	Retries    uint32
}

// Validate validates CreateJobParams.
func (p CreateJobParams) Validate() error {
	switch p.Type {
	case MySQLBackupJob:
	case MySQLRestoreBackupJob:
	case MongoDBBackupJob:
	case MongoDBRestoreBackupJob:
	default:
		return errors.Errorf("unknown job type: %v", p.Type)
	}
	return nil
}

// CreateJob stores a job result in the storage.
func CreateJob(q *reform.Querier, params CreateJobParams) (*Job, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	result := &Job{
		ID:         uuid.New().String(),
		PMMAgentID: params.PMMAgentID,
		Type:       params.Type,
		Data:       params.Data,
		Timeout:    params.Timeout,
		Interval:   params.Interval,
		Retries:    params.Retries,
	}
	if err := q.Insert(result); err != nil {
		return nil, errors.WithStack(err)
	}
	return result, nil
}

// CleanupOldJobs deletes jobs results older than a specified date.
func CleanupOldJobs(q *reform.Querier, olderThan time.Time) error {
	_, err := q.DeleteFrom(JobTable, " WHERE updated_at <= $1", olderThan)
	return err
}

// CreateJobLogParams are params for creating a new job jog.
type CreateJobLogParams struct {
	JobID     string
	ChunkID   int
	Data      string
	LastChunk bool
}

// CreateJobLog inserts new chunk log.
func CreateJobLog(q *reform.Querier, params CreateJobLogParams) (*JobLog, error) {
	log := &JobLog{
		JobID:     params.JobID,
		ChunkID:   params.ChunkID,
		Data:      params.Data,
		LastChunk: params.LastChunk,
	}
	if err := q.Insert(log); err != nil {
		return nil, errors.WithStack(err)
	}
	return log, nil
}

// JobLogsFilter represents filter for job logs.
type JobLogsFilter struct {
	JobID  string
	Offset int
	Limit  *int
}

// FindJobLogs returns logs that belongs to job.
func FindJobLogs(q *reform.Querier, filters JobLogsFilter) ([]*JobLog, error) {
	limit := defaultLimit
	tail := "WHERE job_id = $1 AND chunk_id >= $2 ORDER BY chunk_id LIMIT $3"
	if filters.Limit != nil {
		limit = *filters.Limit
	}
	args := []interface{}{
		filters.JobID,
		filters.Offset,
		limit,
	}

	rows, err := q.SelectAllFrom(JobLogView, tail, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select artifacts")
	}

	logs := make([]*JobLog, 0, len(rows))
	for _, r := range rows {
		logs = append(logs, r.(*JobLog)) //nolint:forcetypeassert
	}
	return logs, nil
}
