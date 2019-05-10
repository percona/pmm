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
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

//go:generate reform

// ServiceType represents Service type as stored in database.
type ServiceType string

// Service types (in the same order as in services.proto).
const (
	MySQLServiceType      ServiceType = "mysql"
	MongoDBServiceType    ServiceType = "mongodb"
	PostgreSQLServiceType ServiceType = "postgresql"
)

// Service represents Service as stored in database.
//reform:services
type Service struct {
	ServiceID    string      `reform:"service_id,pk"`
	ServiceType  ServiceType `reform:"service_type"`
	ServiceName  string      `reform:"service_name"`
	NodeID       string      `reform:"node_id"`
	CustomLabels []byte      `reform:"custom_labels"`
	CreatedAt    time.Time   `reform:"created_at"`
	UpdatedAt    time.Time   `reform:"updated_at"`

	Address *string `reform:"address"`
	Port    *uint16 `reform:"port"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//nolint:unparam
func (s *Service) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//nolint:unparam
func (s *Service) BeforeUpdate() error {
	s.UpdatedAt = Now()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
//nolint:unparam
func (s *Service) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// GetCustomLabels decodes custom labels.
func (s *Service) GetCustomLabels() (map[string]string, error) {
	if len(s.CustomLabels) == 0 {
		return nil, nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(s.CustomLabels, &m); err != nil {
		return nil, errors.Wrap(err, "failed to decode custom labels")
	}
	return m, nil
}

// SetCustomLabels encodes custom labels.
func (s *Service) SetCustomLabels(m map[string]string) error {
	if len(m) == 0 {
		s.CustomLabels = nil
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "failed to encode custom labels")
	}
	s.CustomLabels = b
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*Service)(nil)
	_ reform.BeforeUpdater  = (*Service)(nil)
	_ reform.AfterFinder    = (*Service)(nil)
)
