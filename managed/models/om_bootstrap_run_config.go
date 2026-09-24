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
// It also carries RegisteredAt, which is what stops the stepper's succeeded-run
// sweep from revisiting a run forever. A run reading SUCCEEDED in SEP is not done
// from PMM's side until every one of its hosts is registered with PMM's own
// inventory, and SEP knows nothing about that step, so the record of it has to
// live here. Nil means that work is still outstanding; set means the run is
// finished and the sweep skips it -- and because it is persisted rather than
// held in the stepper, a new leader after a failover, or the same one after a
// restart, skips it too.
//
// Created at trigger time when the caller gave a label to store, and otherwise
// on completion. Absence (ErrNotFound) is expected and not an error: an
// unlabelled run that has not finished registering has no row yet, and reads
// back as an unlabelled service exactly as it always did.
//
//reform:om_bootstrap_run_configs
type OmBootstrapRunConfig struct {
	RunID        string     `reform:"run_id,pk"`
	Environment  string     `reform:"environment"`
	Cluster      string     `reform:"cluster"`
	CreatedAt    time.Time  `reform:"created_at"`
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
