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

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

//go:generate go tool reform

// OmSetting is one Operations policy value, global when ClusterID is nil else a cluster override.
//
//reform:om_settings
type OmSetting struct {
	ID        string    `reform:"id,pk"`
	Key       string    `reform:"key"`
	ClusterID *string   `reform:"cluster_id"`
	Value     []byte    `reform:"value"`
	UpdatedBy string    `reform:"updated_by"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter.
func (s *OmSetting) BeforeInsert() error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = Now()
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater.
func (s *OmSetting) BeforeUpdate() error {
	s.UpdatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder.
func (s *OmSetting) AfterFind() error {
	s.UpdatedAt = s.UpdatedAt.UTC()
	return nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*OmSetting)(nil)
	_ reform.BeforeUpdater  = (*OmSetting)(nil)
	_ reform.AfterFinder    = (*OmSetting)(nil)
)
