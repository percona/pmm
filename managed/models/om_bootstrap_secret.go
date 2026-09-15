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

// OmBootstrapSecret holds the one MongoDB user PMM-15347's om_bootstrap stepper
// generates for a run, and the replica set's shared keyFile content, encrypted at
// rest.
//
// One user, not two: it authenticates both the customer's own administration (it is
// the replica set's very first user, created through MongoDB's localhost exception)
// and PMM's own mongodb_exporter, rather than a separate narrow monitoring account --
// phase-1 scope, see PMM-15347/questions.md Q7.
//
// Keyed by SEP's run ID (a UUID string), not one PMM mints itself: this table exists
// for exactly one reason -- the stepper generates this value once, uses it across
// several dispatches (every host's distribute_keyfile step, create_pmm_monitoring_user,
// and finally the AddService call that registers the mongod with PMM's own
// inventory), and any of those can happen on a *different* pmm-managed leader than
// generated it, after a failover. SEP's run ID is already the one identifier both
// sides agree on for "which bootstrap this is," and om_bootstrap itself never sees
// the plaintext (PMM-15347/plan.md §4 item 9: SEP owns run/step state, not secrets).
//
//reform:om_bootstrap_secrets
type OmBootstrapSecret struct {
	RunID           string    `reform:"run_id,pk"`
	MongoDBUsername string    `reform:"mongodb_username"`
	MongoDBPassword string    `reform:"mongodb_password"`
	KeyFile         string    `reform:"key_file"`
	CreatedAt       time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (s *OmBootstrapSecret) BeforeInsert() error {
	s.CreatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder.
func (s *OmBootstrapSecret) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	return nil
}

// Check that OmBootstrapSecret satisfies reform's hooks.
var (
	_ reform.BeforeInserter = (*OmBootstrapSecret)(nil)
	_ reform.AfterFinder    = (*OmBootstrapSecret)(nil)
)
