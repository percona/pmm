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

// OmBootstrapSecret holds a bootstrap run's MongoDB users and keyFile, encrypted at rest.
//
//reform:om_bootstrap_secrets
type OmBootstrapSecret struct {
	RunID           string          `reform:"run_id,pk"`
	MongoDBUsername EncryptedString `reform:"mongodb_username"`
	MongoDBPassword EncryptedString `reform:"mongodb_password"`
	KeyFile         EncryptedString `reform:"key_file"`
	AdminUsername   EncryptedString `reform:"admin_username"`
	AdminPassword   EncryptedString `reform:"admin_password"`
	CreatedAt       time.Time       `reform:"created_at"`
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
