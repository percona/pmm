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

//go:generate go tool reform

// OmBootstrapRunConfig holds the PMM-only labels and registration state of a PMM Extensions bootstrap run.
//
//reform:om_bootstrap_run_configs
type OmBootstrapRunConfig struct {
	RunID       string    `reform:"run_id,pk"`
	Environment string    `reform:"environment"`
	Cluster     string    `reform:"cluster"`
	CreatedAt   time.Time `reform:"created_at"`
	// RegisteredAt is nil until PMM has finished registering the run's hosts, successfully or not.
	RegisteredAt *time.Time `reform:"registered_at"`
	// RegistrationErrors holds, by executor host, why PMM could not register or see that host's service by then.
	RegistrationErrors OmRegistrationErrors `reform:"registration_errors"`
}

// OmRegistrationErrors maps an executor host to why PMM gave up registering its MongoDB service.
type OmRegistrationErrors map[string]string

// Value implements database/sql/driver.Valuer interface, storing an empty map as NULL.
func (e OmRegistrationErrors) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil //nolint:nilnil // NULL is the stored form of no errors
	}
	return jsonValue(e)
}

// Scan implements database/sql.Scanner interface.
func (e *OmRegistrationErrors) Scan(src any) error { return jsonScan(e, src) }

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
