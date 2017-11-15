// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package models

import (
	"database/sql"
	"database/sql/driver"

	"github.com/pkg/errors"
)

type ServiceType string

const (
	RDSServiceType ServiceType = "rds"
)

func (u ServiceType) Value() (driver.Value, error) {
	return string(u), nil
}

func (u *ServiceType) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		*u = ServiceType(src)
	case []byte:
		*u = ServiceType(src)
	default:
		return errors.Errorf("unexpected type %T (%#v)", src, src)
	}
	return nil
}

// check interfaces
// TODO we should not need those methods with version 1.4 of the MySQL driver, and with SQLite3 driver
var (
	_ driver.Valuer = ServiceType("")
	_ sql.Scanner   = (*ServiceType)(nil)
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

	Address       *string `reform:"address"`
	Port          *uint16 `reform:"port"`
	Engine        *string `reform:"engine"`
	EngineVersion *string `reform:"engine_version"`
}
