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

package models

import (
	"database/sql/driver"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate reform

// JobType represents job type.
type JobType string

// Supported job types.
const (
	Echo                  = JobType("echo")
	MySQLBackupJob        = JobType("mysql_backup")
	MySQLRestoreBackupJob = JobType("mysql_restore_backup")
)

// EchoJobResult stores echo job specific result data.
type EchoJobResult struct {
	Message string `json:"message"`
}

// MySQLBackupJobResult stores MySQL job specific result data.
type MySQLBackupJobResult struct {
	ArtifactID string `json:"artifact_id"`
}

// MySQLRestoreBackupJobResult stores MySQL restore backup job specific result data.
type MySQLRestoreBackupJobResult struct {
	RestoreID string `json:"restore_id,omitempty"`
}

// JobResultData holds result data for different job types.
type JobResultData struct {
	Echo               *EchoJobResult               `json:"echo,omitempty"`
	MySQLBackup        *MySQLBackupJobResult        `json:"mysql_backup,omitempty"`
	MySQLRestoreBackup *MySQLRestoreBackupJobResult `json:"mysql_restore_backup,omitempty"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c JobResultData) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *JobResultData) Scan(src interface{}) error { return jsonScan(c, src) }

// JobResult describes a job result which is storing in persistent storage.
//reform:job_results
type JobResult struct {
	ID         string         `reform:"id,pk"`
	PMMAgentID string         `reform:"pmm_agent_id"`
	Type       JobType        `reform:"type"`
	Done       bool           `reform:"done"`
	Error      string         `reform:"error"`
	Result     *JobResultData `reform:"result"`
	CreatedAt  time.Time      `reform:"created_at"`
	UpdatedAt  time.Time      `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *JobResult) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *JobResult) BeforeUpdate() error {
	r.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *JobResult) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*JobResult)(nil)
	_ reform.BeforeUpdater  = (*JobResult)(nil)
	_ reform.AfterFinder    = (*JobResult)(nil)
)
