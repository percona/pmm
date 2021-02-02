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

// BackupLocationType represents BackupLocation type as stored in database.
type BackupLocationType string

// BackupLocation types
const (
	S3BackupLocationType BackupLocationType = "s3"
	FSBackupLocationType BackupLocationType = "fs"
)

// BackupLocation represents destination for backup.
//reform:backup_locations
type BackupLocation struct {
	ID          string             `reform:"id,pk"`
	Name        string             `reform:"name"`
	Description string             `reform:"description"`
	Type        BackupLocationType `reform:"type"`
	S3Config    *S3LocationConfig  `reform:"s3_config"`
	FSConfig    *FSLocationConfig  `reform:"fs_config"`

	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *BackupLocation) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *BackupLocation) BeforeUpdate() error {
	s.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *BackupLocation) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	return nil
}

// S3LocationConfig contains required properties for accessing S3 Bucket
type S3LocationConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c S3LocationConfig) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *S3LocationConfig) Scan(src interface{}) error { return jsonScan(c, src) }

// FSLocationConfig contains require properties for accessing file system
type FSLocationConfig struct {
	Path string `json:"path"`
}

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (c FSLocationConfig) Value() (driver.Value, error) { return jsonValue(c) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (c *FSLocationConfig) Scan(src interface{}) error { return jsonScan(c, src) }

// check interfaces
var (
	_ reform.BeforeInserter = (*BackupLocation)(nil)
	_ reform.BeforeUpdater  = (*BackupLocation)(nil)
	_ reform.AfterFinder    = (*BackupLocation)(nil)
)
