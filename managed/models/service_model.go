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

//go:generate ../../bin/reform

// ServiceType represents Service type as stored in databases:
// pmm-managed's PostgreSQL, qan-api's ClickHouse, and VictoriaMetrics.
type (
	ServiceType string
	// ServiceStandardLabelsParams represents the parameters for standard labels in a service.
	ServiceStandardLabelsParams struct {
		Cluster        *string
		Environment    *string
		ReplicationSet *string
		ExternalGroup  *string
	}
)

// Service types (in the same order as in services.proto).
const (
	MySQLServiceType      ServiceType = "mysql"
	MongoDBServiceType    ServiceType = "mongodb"
	PostgreSQLServiceType ServiceType = "postgresql"
	ProxySQLServiceType   ServiceType = "proxysql"
	HAProxyServiceType    ServiceType = "haproxy"
	ExternalServiceType   ServiceType = "external"
)

// Service represents Service as stored in database.
//
//reform:services
type Service struct {
	ServiceID      string      `reform:"service_id,pk"`
	ServiceType    ServiceType `reform:"service_type"`
	ServiceName    string      `reform:"service_name"`
	DatabaseName   string      `reform:"database_name"`
	NodeID         string      `reform:"node_id"`
	Environment    string      `reform:"environment"`
	Cluster        string      `reform:"cluster"`
	ReplicationSet string      `reform:"replication_set"`
	CustomLabels   []byte      `reform:"custom_labels"`
	ExternalGroup  string      `reform:"external_group"`
	Version        *string     `reform:"version"`
	CreatedAt      time.Time   `reform:"created_at"`
	UpdatedAt      time.Time   `reform:"updated_at"`

	Address *string `reform:"address"`
	Port    *uint16 `reform:"port"`
	Socket  *string `reform:"socket"`
}

// BeforeInsert implements reform.BeforeInserter interface.
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
func (s *Service) BeforeUpdate() error {
	s.UpdatedAt = Now()
	if len(s.CustomLabels) == 0 {
		s.CustomLabels = nil
	}
	return nil
}

// AfterFind implements reform.AfterFinder interface.
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
	return getLabels(s.CustomLabels)
}

// SetCustomLabels encodes custom labels.
func (s *Service) SetCustomLabels(m map[string]string) error {
	return setLabels(m, &s.CustomLabels)
}

// UnifiedLabels returns combined standard and custom labels with empty labels removed.
func (s *Service) UnifiedLabels() (map[string]string, error) {
	custom, err := s.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	res := map[string]string{
		"service_id":      s.ServiceID,
		"service_name":    s.ServiceName,
		"service_type":    string(s.ServiceType),
		"environment":     s.Environment,
		"cluster":         s.Cluster,
		"replication_set": s.ReplicationSet,
		"external_group":  s.ExternalGroup,
	}
	for name, value := range custom {
		res[name] = value
	}

	if err = prepareLabels(res, true); err != nil {
		return nil, err
	}
	return res, nil
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*Service)(nil)
	_ reform.BeforeUpdater  = (*Service)(nil)
	_ reform.AfterFinder    = (*Service)(nil)
)
