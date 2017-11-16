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

type NodeType string

const (
	PMMServerNodeType NodeType = "pmm-server"
	RDSNodeType       NodeType = "rds"
)

func (u NodeType) Value() (driver.Value, error) {
	return string(u), nil
}

func (u *NodeType) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		*u = NodeType(src)
	case []byte:
		*u = NodeType(src)
	default:
		return errors.Errorf("unexpected type %T (%#v)", src, src)
	}
	return nil
}

// check interfaces
// TODO we should not need those methods with version 1.4 of the MySQL driver, and with SQLite3 driver
var (
	_ driver.Valuer = NodeType("")
	_ sql.Scanner   = (*NodeType)(nil)
)

//reform:nodes
type Node struct {
	ID   int32    `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"`
}

//reform:nodes
type RDSNode struct {
	ID   int32    `reform:"id,pk"`
	Type NodeType `reform:"type"`
	Name string   `reform:"name"` // DBInstanceIdentifier

	Region string `reform:"region"`
}
