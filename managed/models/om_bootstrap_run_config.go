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

// OmBootstrapRunConfig holds the PMM-only configuration a bootstrap run was
// triggered with, that SEP's own om_bootstrap app has no use for and so never
// stores: the environment and cluster to label the resulting MongoDB service
// with once it is registered (see registerBootstrapHost's own doc comment).
// Everything else about a run -- install_method, os, mongodb_version,
// replica_set_name -- SEP already persists and PMM reads back through the
// proxy; these two exist only to survive from TriggerHostBootstrap, which
// receives them from the caller, to completeSucceededRun, which is the only
// place they are ever consumed -- potentially on a different pmm-managed
// leader after a failover, same reasoning as OmBootstrapSecret's own doc
// comment.
//
// Created once, at trigger time, and never updated -- unlike OmBootstrapSecret,
// which is generated lazily on first use because nothing needs a value until
// then. Absence (ErrNotFound) is expected and not an error: a run triggered
// before this table existed, or one whose caller left both fields blank, has
// no row, and reads back as an unlabelled service exactly as it always did.
//
//reform:om_bootstrap_run_configs
type OmBootstrapRunConfig struct {
	RunID       string    `reform:"run_id,pk"`
	Environment string    `reform:"environment"`
	Cluster     string    `reform:"cluster"`
	CreatedAt   time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (c *OmBootstrapRunConfig) BeforeInsert() error {
	c.CreatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder.
func (c *OmBootstrapRunConfig) AfterFind() error {
	c.CreatedAt = c.CreatedAt.UTC()
	return nil
}

// Check that OmBootstrapRunConfig satisfies reform's hooks.
var (
	_ reform.BeforeInserter = (*OmBootstrapRunConfig)(nil)
	_ reform.AfterFinder    = (*OmBootstrapRunConfig)(nil)
)
