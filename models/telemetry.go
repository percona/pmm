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

// TelemetryRow stores telemetry information.
//reform:telemetry
type TelemetryRow struct {
	UUID      string    `reform:"uuid,pk"`
	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//nolint:unparam
func (t *TelemetryRow) BeforeInsert() error {
	if t.UUID == "" {
		return errors.New("UUID should not be empty")
	}

	now := Now()
	t.CreatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//nolint:unparam
func (t *TelemetryRow) BeforeUpdate() error {
	panic("TelemetryRow should not be updated")
}

// AfterFind implements reform.AfterFinder interface.
//nolint:unparam
func (t *TelemetryRow) AfterFind() error {
	t.CreatedAt = t.CreatedAt.UTC()
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*TelemetryRow)(nil)
	_ reform.BeforeUpdater  = (*TelemetryRow)(nil)
	_ reform.AfterFinder    = (*TelemetryRow)(nil)
)
