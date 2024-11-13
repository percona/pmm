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

// DataModel represents a data model used for performing a backup.
type DataModel string

// DataModel types (in the same order as in artifacts.proto).
const (
	PhysicalDataModel DataModel = "physical"
	LogicalDataModel  DataModel = "logical"
)

// Validate validates data model.
func (dm DataModel) Validate() error {
	switch dm {
	case PhysicalDataModel:
	case LogicalDataModel:
	case "":
		return NewInvalidArgumentError("empty data model")
	default:
		return NewInvalidArgumentError("invalid data model '%s'", dm)
	}

	return nil
}

// BackupStatus shows current status of backup.
type BackupStatus string

// BackupStatus status (in the same order as in artifacts.proto).
const (
	PendingBackupStatus        BackupStatus = "pending"
	InProgressBackupStatus     BackupStatus = "in_progress"
	PausedBackupStatus         BackupStatus = "paused"
	SuccessBackupStatus        BackupStatus = "success"
	ErrorBackupStatus          BackupStatus = "error"
	DeletingBackupStatus       BackupStatus = "deleting"
	FailedToDeleteBackupStatus BackupStatus = "failed_to_delete"
	CleanupInProgressStatus    BackupStatus = "cleanup_in_progress"
)

// Validate validates backup status.
func (bs BackupStatus) Validate() error {
	switch bs {
	case PendingBackupStatus:
	case InProgressBackupStatus:
	case PausedBackupStatus:
	case SuccessBackupStatus:
	case ErrorBackupStatus:
	case DeletingBackupStatus:
	case FailedToDeleteBackupStatus:
	default:
		return NewInvalidArgumentError("invalid status '%s'", bs)
	}

	return nil
}

// Pointer returns a pointer to status value.
func (bs BackupStatus) Pointer() *BackupStatus {
	return &bs
}

// ArtifactType represents type how artifact was created.
type ArtifactType string

// ArtifactType types.
const (
	OnDemandArtifactType  ArtifactType = "on_demand"
	ScheduledArtifactType ArtifactType = "scheduled"
)

// BackupMode represents artifact mode.
type BackupMode string

// Backup modes.
const (
	Snapshot    BackupMode = "snapshot"
	Incremental BackupMode = "incremental"
	PITR        BackupMode = "pitr"
)

// Validate validates backup mode.
func (m BackupMode) Validate() error {
	switch m {
	case Snapshot:
	case Incremental:
	case PITR:
	case "":
		return NewInvalidArgumentError("empty backup mode")
	default:
		return NewInvalidArgumentError("invalid backup mode '%s'", m)
	}

	return nil
}

// File represents file or directory.
type File struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"is_directory"`
}

// PbmMetadata contains extra data for pbm cli tool.
type PbmMetadata struct {
	// Name of backup in pbm representation.
	Name string `json:"name"`
}

// BackupToolData contains extra data for backup tools.
type BackupToolData struct {
	PbmMetadata *PbmMetadata
}

// Metadata contains extra artifact data like files it consists of, tool specific data, etc.
type Metadata struct {
	FileList       []File          `json:"file_list"`
	RestoreTo      *time.Time      `json:"restore_to"`
	BackupToolData *BackupToolData `json:"backup_tool_data"`
}

// MetadataList is a list of metadata associated with artifacts.
type MetadataList []Metadata

// Value implements database/sql/driver.Valuer interface. Should be defined on the value.
func (p MetadataList) Value() (driver.Value, error) { return jsonValue(p) }

// Scan implements database/sql.Scanner interface. Should be defined on the pointer.
func (p *MetadataList) Scan(src interface{}) error { return jsonScan(p, src) }

// Artifact represents result of a backup.
//
//reform:artifacts
type Artifact struct {
	ID               string       `reform:"id,pk"`
	Name             string       `reform:"name"`
	Vendor           string       `reform:"vendor"`
	DBVersion        string       `reform:"db_version"`
	LocationID       string       `reform:"location_id"`
	ServiceID        string       `reform:"service_id"`
	DataModel        DataModel    `reform:"data_model"`
	Mode             BackupMode   `reform:"mode"`
	Status           BackupStatus `reform:"status"`
	Type             ArtifactType `reform:"type"`
	ScheduleID       string       `reform:"schedule_id"`
	CreatedAt        time.Time    `reform:"created_at"`
	UpdatedAt        time.Time    `reform:"updated_at"`
	IsShardedCluster bool         `reform:"is_sharded_cluster"`
	Folder           string       `reform:"folder"`
	MetadataList     MetadataList `reform:"metadata_list"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *Artifact) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *Artifact) BeforeUpdate() error {
	s.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *Artifact) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Artifact)(nil)
	_ reform.BeforeUpdater  = (*Artifact)(nil)
	_ reform.AfterFinder    = (*Artifact)(nil)
)
