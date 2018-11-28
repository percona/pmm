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

type ServiceType string

// Service types.
const (
	MySQLServiceType ServiceType = "mysql"

	AWSRDSServiceType     ServiceType = "aws-rds"
	PostgreSQLServiceType ServiceType = "postgresql"
)

//reform:services
type Service struct {
	ID     uint32      `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	NodeID uint32      `reform:"node_id"`
}

//reform:services
type ServiceRow struct {
	ID        uint32      `reform:"id,pk"`
	Type      ServiceType `reform:"type"`
	Name      string      `reform:"name"`
	NodeID    uint32      `reform:"node_id"`
	CreatedAt time.Time   `reform:"created_at"`
	UpdatedAt time.Time   `reform:"updated_at"`

	Address    *string `reform:"address"`
	Port       *uint16 `reform:"port"`
	UnixSocket *string `reform:"unix_socket"`
}

func (sr *ServiceRow) BeforeInsert() error {
	now := time.Now().Truncate(time.Microsecond).UTC()
	sr.CreatedAt = now
	sr.UpdatedAt = now
	return nil
}

func (sr *ServiceRow) BeforeUpdate() error {
	now := time.Now().Truncate(time.Microsecond).UTC()
	sr.UpdatedAt = now
	return nil
}

func (sr *ServiceRow) AfterFind() error {
	sr.CreatedAt = sr.CreatedAt.UTC()
	sr.UpdatedAt = sr.UpdatedAt.UTC()
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*ServiceRow)(nil)
	_ reform.BeforeUpdater  = (*ServiceRow)(nil)
	_ reform.AfterFinder    = (*ServiceRow)(nil)
)

// TODO remove types below

//reform:services
type AWSRDSService struct {
	ID     uint32      `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	Name   string      `reform:"name"`
	NodeID uint32      `reform:"node_id"`

	AWSAccessKey  *string `reform:"aws_access_key"` // may be nil
	AWSSecretKey  *string `reform:"aws_secret_key"` // may be nil
	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}

//reform:services
type PostgreSQLService struct {
	ID     uint32      `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	Name   string      `reform:"name"`
	NodeID uint32      `reform:"node_id"`

	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}

//reform:services
type MySQLService struct {
	ID     uint32      `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	Name   string      `reform:"name"`
	NodeID uint32      `reform:"node_id"`

	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}

//reform:services
type RemoteService struct {
	ID     uint32      `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	Name   string      `reform:"name"`
	NodeID uint32      `reform:"node_id"`

	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}
