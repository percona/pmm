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
	"database/sql/driver"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// JobType represents job type.
type JobType string

// Supported job types.
const (
	MySQLBackupJob          = JobType("mysql_backup")
	MySQLRestoreBackupJob   = JobType("mysql_restore_backup")
	MongoDBBackupJob        = JobType("mongodb_backup")
	MongoDBRestoreBackupJob = JobType("mongodb_restore_backup")
)

// MySQLBackupJobResult stores MySQL job specific result data.
type MySQLBackupJobResult struct{}

// MySQLRestoreBackupJobResult stores MySQL restore backup job specific result data.
type MySQLRestoreBackupJobResult struct{}

// MongoDBBackupJobResult stores MongoDB job specific result data.
type MongoDBBackupJobResult struct{}

// MongoDBRestoreBackupJobResult stores MongoDB restore backup job specific result data.
type MongoDBRestoreBackupJobResult struct{}

// JobResult holds result data for different job types.
type JobResult struct {
	MySQLBackup          *MySQLBackupJobResult          `json:"mysql_backup,omitempty"`
	MySQLRestoreBackup   *MySQLRestoreBackupJobResult   `json:"mysql_restore_backup,omitempty"`
	MongoDBBackup        *MongoDBBackupJobResult        `json:"mongo_db_backup,omitempty"`
	MongoDBRestoreBackup *MongoDBRestoreBackupJobResult `json:"mongo_db_restore_backup,omitempty"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (r JobResult) Value() (driver.Value, error) { return jsonValue(r) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (r *JobResult) Scan(src interface{}) error { return jsonScan(r, src) }

// MySQLBackupJobData stores MySQL job specific result data.
type MySQLBackupJobData struct {
	ServiceID  string `json:"service_id"`
	ArtifactID string `json:"artifact_id"`
}

// MySQLRestoreBackupJobData stores MySQL restore backup job specific result data.
type MySQLRestoreBackupJobData struct {
	ServiceID string `json:"service_id"`
	RestoreID string `json:"restore_id"`
}

// MongoDBBackupJobData stores MongoDB job specific result data.
type MongoDBBackupJobData struct {
	ServiceID  string     `json:"service_id"`
	ArtifactID string     `json:"artifact_id"`
	Mode       BackupMode `json:"mode"`
	DataModel  DataModel  `json:"data_model"`
}

// MongoDBRestoreBackupJobData stores MongoDB restore backup job specific result data.
type MongoDBRestoreBackupJobData struct {
	ServiceID string    `json:"service_id"`
	RestoreID string    `json:"restore_id"`
	DataModel DataModel `json:"data_model"`
}

// JobData contains data required for running a job.
type JobData struct {
	MySQLBackup          *MySQLBackupJobData          `json:"mysql_backup,omitempty"`
	MySQLRestoreBackup   *MySQLRestoreBackupJobData   `json:"mysql_restore_backup,omitempty"`
	MongoDBBackup        *MongoDBBackupJobData        `json:"mongodb_backup,omitempty"`
	MongoDBRestoreBackup *MongoDBRestoreBackupJobData `json:"mongodb_restore_backup,omitempty"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c JobData) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *JobData) Scan(src interface{}) error { return jsonScan(c, src) }

// Job describes a job result which is storing in persistent storage.
//
//reform:jobs
type Job struct {
	ID         string        `reform:"id,pk"`
	PMMAgentID string        `reform:"pmm_agent_id"`
	Type       JobType       `reform:"type"`
	Data       *JobData      `reform:"data"`
	Timeout    time.Duration `reform:"timeout"`
	Retries    uint32        `reform:"retries"`
	Interval   time.Duration `reform:"interval"`
	Done       bool          `reform:"done"`
	Error      string        `reform:"error"`
	Result     *JobResult    `reform:"result"`
	CreatedAt  time.Time     `reform:"created_at"`
	UpdatedAt  time.Time     `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (r *Job) BeforeInsert() error {
	now := Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (r *Job) BeforeUpdate() error {
	r.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (r *Job) AfterFind() error {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()

	return nil
}

// JobLog stores chunk of logs from job.
//
//reform:job_logs
type JobLog struct {
	JobID     string `reform:"job_id"`
	ChunkID   int    `reform:"chunk_id"`
	Data      string `reform:"data"`
	LastChunk bool   `reform:"last_chunk"`
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Job)(nil)
	_ reform.BeforeUpdater  = (*Job)(nil)
	_ reform.AfterFinder    = (*Job)(nil)
)
