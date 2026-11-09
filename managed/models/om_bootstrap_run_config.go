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

// OmBootstrapRunConfig holds the PMM-only labels and registration state of a PMM Extensions bootstrap run.
//
//reform:om_bootstrap_run_configs
type OmBootstrapRunConfig struct {
	RunID       string    `reform:"run_id,pk"`
	Environment string    `reform:"environment"`
	Cluster     string    `reform:"cluster"`
	CreatedAt   time.Time `reform:"created_at"`
	// RegisteredAt is nil until every host of the run is registered in PMM's inventory.
	RegisteredAt *time.Time `reform:"registered_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (c *OmBootstrapRunConfig) BeforeInsert() error {
	c.CreatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder.
func (c *OmBootstrapRunConfig) AfterFind() error {
	c.CreatedAt = c.CreatedAt.UTC()
	if c.RegisteredAt != nil {
		registeredAt := c.RegisteredAt.UTC()
		c.RegisteredAt = &registeredAt
	}
	return nil
}

// Check that OmBootstrapRunConfig satisfies reform's hooks.
var (
	_ reform.BeforeInserter = (*OmBootstrapRunConfig)(nil)
	_ reform.AfterFinder    = (*OmBootstrapRunConfig)(nil)
)
