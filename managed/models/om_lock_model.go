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

//go:generate go tool reform

// OmLockMode is a lock's strength. Exclusive conflicts with any other holder, shared only
// with an exclusive one.
type OmLockMode string

// Lock modes.
const (
	OmLockShared    = OmLockMode("shared")
	OmLockExclusive = OmLockMode("exclusive")
)

// OmLock is a held lock; rows are written by om_acquire_locks, so this is the read surface.
//
//reform:om_locks
type OmLock struct {
	ResourceKind string     `reform:"resource_kind"`
	ResourceID   string     `reform:"resource_id"`
	OperationID  string     `reform:"operation_id"`
	Mode         OmLockMode `reform:"mode"`
	AcquiredAt   time.Time  `reform:"acquired_at"`
}

// AfterFind implements reform.AfterFinder.
func (l *OmLock) AfterFind() error {
	l.AcquiredAt = l.AcquiredAt.UTC()
	return nil
}

// OmLockResource serialises acquisitions on one resource and is trigger-written.
//
//reform:om_lock_resources
type OmLockResource struct {
	ResourceKind string `reform:"resource_kind"`
	ResourceID   string `reform:"resource_id"`
	Version      int64  `reform:"version"`
}

// OmLeaderLease is the singleton Operations leader epoch, advanced by om_acquire_leader_lease.
//
//reform:om_leader_lease
type OmLeaderLease struct {
	Singleton bool       `reform:"singleton"`
	Epoch     int64      `reform:"epoch"`
	Holder    *string    `reform:"holder"`
	ExpiresAt *time.Time `reform:"expires_at"`
}

// AfterFind implements reform.AfterFinder.
func (l *OmLeaderLease) AfterFind() error {
	if l.ExpiresAt != nil {
		t := l.ExpiresAt.UTC()
		l.ExpiresAt = &t
	}
	return nil
}

// check interfaces.
var (
	_ reform.AfterFinder = (*OmLock)(nil)
	_ reform.AfterFinder = (*OmLeaderLease)(nil)
)
