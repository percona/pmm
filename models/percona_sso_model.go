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

	"gopkg.in/reform.v1"
)

//go:generate reform

// PerconaSSODetails stores everything we need to issue access_token from Percona SSO API.
// It is intended to have only one row in this table as PMM can be connected to Portal only once.
//reform:percona_sso_details
type PerconaSSODetails struct {
	ClientID     string `reform:"client_id"`
	ClientSecret string `reform:"client_secret"`
	IssuerURL    string `reform:"issuer_url"`
	Scope        string `reform:"scope"`

	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *PerconaSSODetails) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*PerconaSSODetails)(nil)
)
