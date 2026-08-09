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
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// Interval represents check execution interval.
type Interval string

// Available check execution intervals.
const (
	Standard Interval = "standard"
	Frequent Interval = "frequent"
	Rare     Interval = "rare"
)

// CheckSource represents the origin of an advisor check.
type CheckSource string

// Available advisor check sources.
const (
	// BuiltinCheckSource is a built-in check shipped with PMM, reconciled from disk at startup.
	BuiltinCheckSource = CheckSource("builtin")
	// UserCheckSource is a user-authored check created via the API.
	UserCheckSource = CheckSource("user")
)

// AdvisorCheck represents an advisor check stored in the database.
// Percona-shipped checks are reconciled from disk into this table at startup;
// user-authored checks are created via the API. Interval overrides and
// enable/disable state (global and per-service) live in dedicated columns so
// that a content refresh never touches them.
//
//reform:advisor_checks
type AdvisorCheck struct {
	Name        string      `reform:"name,pk"`
	Source      CheckSource `reform:"source"`
	Version     uint32      `reform:"version"`
	Summary     string      `reform:"summary"`
	Description string      `reform:"description"`
	Category    string      `reform:"category"`
	Technology  string      `reform:"technology"`
	// Interval is the original author-defined execution interval.
	Interval string `reform:"interval"`
	// IntervalOverride is the user-set execution interval; nil means no override.
	IntervalOverride *string `reform:"interval_override"`
	// Disabled reports whether the check is disabled globally.
	Disabled bool `reform:"disabled"`
	// DisabledServiceIDs holds a JSON-encoded array of service IDs for which
	// the check is disabled; nil means none.
	DisabledServiceIDs []byte `reform:"disabled_service_ids"`
	// Queries holds the JSON-encoded []check.Query.
	Queries   []byte    `reform:"queries"`
	Script    string    `reform:"script"`
	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// GetDisabledServiceIDs decodes the list of service IDs for which the check is disabled.
func (c *AdvisorCheck) GetDisabledServiceIDs() ([]string, error) {
	if len(c.DisabledServiceIDs) == 0 {
		return nil, nil
	}

	var ids []string
	err := json.Unmarshal(c.DisabledServiceIDs, &ids)
	if err != nil {
		return nil, fmt.Errorf("failed to decode disabled service IDs: %w", err)
	}
	return ids, nil
}

// SetDisabledServiceIDs encodes the list of service IDs for which the check is disabled.
func (c *AdvisorCheck) SetDisabledServiceIDs(ids []string) error {
	if len(ids) == 0 {
		c.DisabledServiceIDs = nil
		return nil
	}

	b, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("failed to encode disabled service IDs: %w", err)
	}
	c.DisabledServiceIDs = b
	return nil
}

// BeforeInsert implements reform.BeforeInserter interface.
func (c *AdvisorCheck) BeforeInsert() error {
	now := Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (c *AdvisorCheck) BeforeUpdate() error {
	c.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (c *AdvisorCheck) AfterFind() error {
	c.CreatedAt = c.CreatedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*AdvisorCheck)(nil)
	_ reform.BeforeUpdater  = (*AdvisorCheck)(nil)
	_ reform.AfterFinder    = (*AdvisorCheck)(nil)
)
