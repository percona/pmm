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

//go:generate reform

type ServiceType string

const (
	RDSServiceType        ServiceType = "rds"
	PostgreSQLServiceType ServiceType = "postgresql"
)

//reform:services
type Service struct {
	ID     int32       `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	NodeID int32       `reform:"node_id"`
}

//reform:services
type RDSService struct {
	ID     int32       `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	NodeID int32       `reform:"node_id"`

	AWSAccessKey  *string `reform:"aws_access_key"` // may be nil
	AWSSecretKey  *string `reform:"aws_secret_key"` // may be nil
	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}

//reform:services
type PostgreSQLService struct {
	ID     int32       `reform:"id,pk"`
	Type   ServiceType `reform:"type"`
	NodeID int32       `reform:"node_id"`

	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}
