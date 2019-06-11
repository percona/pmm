// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package actions

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/pmm/api/agentpb"
)

type mysqlShowIndexAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLShowIndexParams
}

// NewMySQLShowIndexAction creates MySQL SHOW INDEX Action.
// This is an Action that can run `SHOW INDEX` command on MySQL service with given DSN.
func NewMySQLShowIndexAction(id string, params *agentpb.StartActionRequest_MySQLShowIndexParams) Action {
	return &mysqlShowIndexAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (e *mysqlShowIndexAction) ID() string {
	return e.id
}

// Type returns an Action type.
func (e *mysqlShowIndexAction) Type() string {
	return "mysql-show-index"
}

// Run runs an Action and returns output and error.
func (e *mysqlShowIndexAction) Run(ctx context.Context) ([]byte, error) {
	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).

	db, err := sql.Open("mysql", e.params.Dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, fmt.Sprintf("SHOW /* pmm-agent */ INDEX IN %#q", e.params.Table))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}

func (e *mysqlShowIndexAction) sealed() {}
