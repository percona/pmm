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

	"gopkg.in/reform.v1"
)

//go:generate ../../bin/reform

// RestoreStatus shows current status of restore.
type RestoreStatus string

// RestoreStatus status (in the same order as in restores.proto).
const (
	InProgressRestoreStatus RestoreStatus = "in_progress"
	SuccessRestoreStatus    RestoreStatus = "success"
	ErrorRestoreStatus      RestoreStatus = "error"
)

// Validate validates restore status.
func (rs RestoreStatus) Validate() error {
	switch rs {
	case InProgressRestoreStatus:
	case SuccessRestoreStatus:
	case ErrorRestoreStatus:
	default:
		return NewInvalidArgumentError("invalid status %q", rs)
	}

	return nil
}

// Pointer returns a pointer to status value.
func (rs RestoreStatus) Pointer() *RestoreStatus {
	return &rs
}

// RestoreHistoryItem represents a restore backup history.
//
//reform:restore_history
type RestoreHistoryItem struct {
	ID            string        `reform:"id,pk"`
	ArtifactID    string        `reform:"artifact_id"`
	ServiceID     string        `reform:"service_id"`
	PITRTimestamp *time.Time    `reform:"pitr_timestamp"`
	Status        RestoreStatus `reform:"status"`
	StartedAt     time.Time     `reform:"started_at"`
	FinishedAt    *time.Time    `reform:"finished_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *RestoreHistoryItem) BeforeInsert() error {
	s.StartedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *RestoreHistoryItem) AfterFind() error {
	s.StartedAt = s.StartedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*RestoreHistoryItem)(nil)
	_ reform.AfterFinder    = (*RestoreHistoryItem)(nil)
)
