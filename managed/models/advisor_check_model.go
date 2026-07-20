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

// AdvisorCheck represents a user-authored advisor check stored in the database.
// Percona-shipped checks are loaded from disk and are not stored here.
//
//reform:advisor_checks
type AdvisorCheck struct {
	Name        string `reform:"name,pk"`
	Summary     string `reform:"summary"`
	Description string `reform:"description"`
	Category    string `reform:"category"`
	Subcategory string `reform:"subcategory"`
	Family      string `reform:"family"`
	Interval    string `reform:"interval"`
	// Queries holds the JSON-encoded []check.Query.
	Queries   []byte    `reform:"queries"`
	Script    string    `reform:"script"`
	CreatedAt time.Time `reform:"created_at"`
	UpdatedAt time.Time `reform:"updated_at"`
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
