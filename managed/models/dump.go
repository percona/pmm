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
	"time"

	"github.com/lib/pq"
	"gopkg.in/reform.v1"
)

// DumpStatus represents the status of a dump process.
//
//go:generate ../../bin/reform
type DumpStatus string

const (
	DumpStatusInProgress = DumpStatus("in_progress") //nolint:revive
	DumpStatusSuccess    = DumpStatus("success")     //nolint:revive
	DumpStatusError      = DumpStatus("error")       //nolint:revive
)

// Validate validates Dumps status.
func (ds DumpStatus) Validate() error {
	switch ds {
	case DumpStatusInProgress:
	case DumpStatusSuccess:
	case DumpStatusError:
	default:
		return NewInvalidArgumentError("invalid dump status '%s'", ds)
	}

	return nil
}

// Pointer returns a pointer to status value.
func (ds DumpStatus) Pointer() *DumpStatus {
	return &ds
}

// Dump represents pmm dump artifact.
//
//reform:dumps
type Dump struct {
	ID           string         `reform:"id,pk"`
	Status       DumpStatus     `reform:"status"`
	ServiceNames pq.StringArray `reform:"service_names"`
	StartTime    *time.Time     `reform:"start_time"`
	EndTime      *time.Time     `reform:"end_time"`
	ExportQAN    bool           `reform:"export_qan"`
	IgnoreLoad   bool           `reform:"ignore_load"`
	CreatedAt    time.Time      `reform:"created_at"`
	UpdatedAt    time.Time      `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (d *Dump) BeforeInsert() error {
	now := Now()
	d.CreatedAt = now
	d.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (d *Dump) BeforeUpdate() error {
	d.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (d *Dump) AfterFind() error {
	d.CreatedAt = d.CreatedAt.UTC()
	d.UpdatedAt = d.UpdatedAt.UTC()
	return nil
}

// DumpLog stores chunk of logs from pmm-dump.
//
//reform:dump_logs
type DumpLog struct {
	DumpID    string `reform:"dump_id"`
	ChunkID   uint32 `reform:"chunk_id"`
	Data      string `reform:"data"`
	LastChunk bool   `reform:"last_chunk"`
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Dump)(nil)
	_ reform.BeforeUpdater  = (*Dump)(nil)
	_ reform.AfterFinder    = (*Dump)(nil)
)
