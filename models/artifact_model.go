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
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

//go:generate reform

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
	default:
		return errors.Wrapf(ErrInvalidArgument, "invalid data model '%s'", dm)
	}

	return nil
}

// BackupStatus shows current status of backup.
type BackupStatus string

// BackupStatus status (in the same order as in artifacts.proto).
const (
	PendingBackupStatus    BackupStatus = "pending"
	InProgressBackupStatus BackupStatus = "in_progress"
	PausedBackupStatus     BackupStatus = "paused"
	SuccessBackupStatus    BackupStatus = "success"
	ErrorBackupStatus      BackupStatus = "error"
)

// Validate validates backup status.
func (bs BackupStatus) Validate() error {
	switch bs {
	case PendingBackupStatus:
	case InProgressBackupStatus:
	case PausedBackupStatus:
	case SuccessBackupStatus:
	case ErrorBackupStatus:
	default:
		return errors.Wrapf(ErrInvalidArgument, "invalid status '%s'", bs)
	}

	return nil
}

// Pointer returns a pointer of backup status.
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

// Artifact represents result of a backup.
//reform:artifacts
type Artifact struct {
	ID         string       `reform:"id,pk"`
	Name       string       `reform:"name"`
	Vendor     string       `reform:"vendor"`
	LocationID string       `reform:"location_id"`
	ServiceID  string       `reform:"service_id"`
	DataModel  DataModel    `reform:"data_model"`
	Status     BackupStatus `reform:"status"`
	Type       ArtifactType `reform:"type"`
	ScheduleID string       `reform:"schedule_id"`
	CreatedAt  time.Time    `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *Artifact) BeforeInsert() error {
	s.CreatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *Artifact) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Artifact)(nil)
	_ reform.AfterFinder    = (*Artifact)(nil)
)
