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
	"time"

	"gopkg.in/reform.v1"
)

//go:generate $PMM_RELEASE_PATH/reform

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

// BackupStatusPointer returns a pointer of backup status.
func BackupStatusPointer(status BackupStatus) *BackupStatus {
	return &status
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

// Artifact represents result of a backup.
//reform:artifacts
type Artifact struct {
	ID         string       `reform:"id,pk"`
	Name       string       `reform:"name"`
	Vendor     string       `reform:"vendor"`
	DBVersion  string       `reform:"db_version"`
	LocationID string       `reform:"location_id"`
	ServiceID  string       `reform:"service_id"`
	DataModel  DataModel    `reform:"data_model"`
	Mode       BackupMode   `reform:"mode"`
	Status     BackupStatus `reform:"status"`
	Type       ArtifactType `reform:"type"`
	ScheduleID string       `reform:"schedule_id"`
	CreatedAt  time.Time    `reform:"created_at"`
	UpdatedAt  time.Time    `reform:"updated_at"`
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
