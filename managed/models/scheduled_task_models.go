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

// ScheduledTaskType represents scheduled task type.
type ScheduledTaskType string

// Supported scheduled task types.
const (
	ScheduledMySQLBackupTask   = ScheduledTaskType("mysql_backup")
	ScheduledMongoDBBackupTask = ScheduledTaskType("mongodb_backup")
)

// ScheduledTask describes a scheduled task.
//
//reform:scheduled_tasks
type ScheduledTask struct {
	ID             string             `reform:"id,pk"`
	CronExpression string             `reform:"cron_expression"`
	Disabled       bool               `reform:"disabled"`
	StartAt        time.Time          `reform:"start_at"`
	LastRun        time.Time          `reform:"last_run"`
	NextRun        time.Time          `reform:"next_run"`
	Type           ScheduledTaskType  `reform:"type"`
	Data           *ScheduledTaskData `reform:"data"`
	Running        bool               `reform:"running"`
	Error          string             `reform:"error"`
	CreatedAt      time.Time          `reform:"created_at"`
	UpdatedAt      time.Time          `reform:"updated_at"`
}

// ScheduledTaskData contains result data for different task types.
type ScheduledTaskData struct {
	MySQLBackupTask   *MySQLBackupTaskData `json:"mysql_backup,omitempty"`
	MongoDBBackupTask *MongoBackupTaskData `json:"mongodb_backup,omitempty"`
}

// CommonBackupTaskData contains common data for all backup tasks.
type CommonBackupTaskData struct {
	ServiceID     string            `json:"service_id"`
	ClusterName   string            `json:"cluster_name"`
	LocationID    string            `json:"location_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Retention     uint32            `json:"retention"`
	DataModel     DataModel         `json:"data_model"`
	Mode          BackupMode        `json:"mode"`
	Retries       uint32            `json:"retries"`
	RetryInterval time.Duration     `json:"retry_interval"`
	Folder        string            `json:"folder"`
	Compression   BackupCompression `json:"compression"`
}

// MySQLBackupTaskData contains data for mysql backup task.
type MySQLBackupTaskData struct {
	CommonBackupTaskData
}

// MongoBackupTaskData contains data for mysql backup task.
type MongoBackupTaskData struct {
	CommonBackupTaskData
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c ScheduledTaskData) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *ScheduledTaskData) Scan(src interface{}) error { return jsonScan(c, src) }

// BeforeInsert implements reform.BeforeInserter interface.
func (s *ScheduledTask) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *ScheduledTask) BeforeUpdate() error {
	s.UpdatedAt = Now()

	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *ScheduledTask) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	s.StartAt = s.StartAt.UTC()
	s.NextRun = s.NextRun.UTC()
	s.LastRun = s.LastRun.UTC()

	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*ScheduledTask)(nil)
	_ reform.BeforeUpdater  = (*ScheduledTask)(nil)
	_ reform.AfterFinder    = (*ScheduledTask)(nil)
)
