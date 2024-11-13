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

func checkUniqueDumpID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty dump ID")
	}

	dump := &Dump{ID: id}
	err := q.Reload(dump)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Dump with id %q already exists.", id)
}

// DumpFilters represents filters for dumps list.
type DumpFilters struct {
	// Return only dumps by specified status.
	Status DumpStatus
}

// CreateDumpParams represents the parameters for creating a dump.
type CreateDumpParams struct {
	ServiceNames []string
	StartTime    *time.Time
	EndTime      *time.Time
	ExportQAN    bool
	IgnoreLoad   bool
}

// Validate checks the validity of CreateDumpParams.
func (p *CreateDumpParams) Validate() error {
	if p.StartTime != nil && p.EndTime != nil && p.StartTime.After(*p.EndTime) {
		return errors.Errorf("dump start time can't be greater than end time")
	}

	return nil
}

// CreateDump creates a dump using the specified parameters.
func CreateDump(q *reform.Querier, params CreateDumpParams) (*Dump, error) {
	if err := params.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid dump creation params")
	}

	id := uuid.New().String()
	if err := checkUniqueDumpID(q, id); err != nil {
		return nil, err
	}

	dump := &Dump{
		ID:           id,
		Status:       DumpStatusInProgress,
		ServiceNames: params.ServiceNames,
		StartTime:    params.StartTime,
		EndTime:      params.EndTime,
		ExportQAN:    params.ExportQAN,
		IgnoreLoad:   params.IgnoreLoad,
	}
	if err := q.Insert(dump); err != nil {
		return nil, errors.WithStack(err)
	}

	return dump, nil
}

// FindDumps returns dumps list sorted by creation time in DESCENDING order.
func FindDumps(q *reform.Querier, filters DumpFilters) ([]*Dump, error) {
	var conditions []string
	var args []interface{}
	var idx int

	if filters.Status != "" {
		idx++
		conditions = append(conditions, fmt.Sprintf("status = %s", q.Placeholder(idx)))
		args = append(args, filters.Status)
	}

	var whereClause string
	if len(conditions) != 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	rows, err := q.SelectAllFrom(DumpTable, fmt.Sprintf("%s ORDER BY created_at DESC", whereClause), args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select dumps")
	}

	dumps := make([]*Dump, 0, len(rows))
	for _, r := range rows {
		dumps = append(dumps, r.(*Dump)) //nolint:forcetypeassert
	}

	return dumps, nil
}

// FindDumpsByIDs finds dumps by IDs.
func FindDumpsByIDs(q *reform.Querier, ids []string) (map[string]*Dump, error) {
	if len(ids) == 0 {
		return make(map[string]*Dump), nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE id IN (%s)", p)
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	all, err := q.SelectAllFrom(DumpTable, tail, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dumps := make(map[string]*Dump, len(all))
	for _, l := range all {
		dump := l.(*Dump) //nolint:forcetypeassert
		dumps[dump.ID] = dump
	}
	return dumps, nil
}

// FindDumpByID returns dump by given ID if found, ErrNotFound if not.
func FindDumpByID(q *reform.Querier, id string) (*Dump, error) {
	if id == "" {
		return nil, errors.New("provided dump id is empty")
	}

	dump := &Dump{ID: id}
	err := q.Reload(dump)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, errors.Wrapf(ErrNotFound, "dump by id '%s'", id)
		}
		return nil, errors.WithStack(err)
	}

	return dump, nil
}

// UpdateDumpStatus updates the status of a dump with the given ID.
func UpdateDumpStatus(q *reform.Querier, id string, status DumpStatus) error {
	dump, err := FindDumpByID(q, id)
	if err != nil {
		return err
	}

	dump.Status = status

	if err = q.Update(dump); err != nil {
		return errors.Wrap(err, "failed to update dump status")
	}

	return nil
}

// DeleteDump removes dump by ID.
func DeleteDump(q *reform.Querier, id string) error {
	if _, err := FindDumpByID(q, id); err != nil {
		return err
	}

	if err := q.Delete(&Dump{ID: id}); err != nil {
		return errors.Wrapf(err, "failed to delete dump by id '%s'", id)
	}
	return nil
}

// CreateDumpLogParams are params for creating a new pmm-dump log.
type CreateDumpLogParams struct {
	DumpID    string
	ChunkID   uint32
	Data      string
	LastChunk bool
}

// CreateDumpLog inserts new chunk log.
func CreateDumpLog(q *reform.Querier, params CreateDumpLogParams) (*DumpLog, error) {
	log := &DumpLog{
		DumpID:    params.DumpID,
		ChunkID:   params.ChunkID,
		Data:      params.Data,
		LastChunk: params.LastChunk,
	}
	if err := q.Insert(log); err != nil {
		return nil, errors.WithStack(err)
	}
	return log, nil
}

// DumpLogsFilter represents filter for dump logs.
type DumpLogsFilter struct {
	DumpID string
	Offset int
	Limit  *int
}

// FindDumpLogs returns logs that belongs to dump.
func FindDumpLogs(q *reform.Querier, filters DumpLogsFilter) ([]*DumpLog, error) {
	limit := defaultLimit
	tail := "WHERE dump_id = $1 AND chunk_id >= $2 ORDER BY chunk_id LIMIT $3"
	if filters.Limit != nil {
		limit = *filters.Limit
	}
	args := []interface{}{
		filters.DumpID,
		filters.Offset,
		limit,
	}

	rows, err := q.SelectAllFrom(DumpLogView, tail, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select dump logs")
	}

	logs := make([]*DumpLog, 0, len(rows))
	for _, r := range rows {
		logs = append(logs, r.(*DumpLog)) //nolint:forcetypeassert
	}
	return logs, nil
}
