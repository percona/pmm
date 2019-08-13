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

type mysqlShowCreateTableAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLShowCreateTableParams
}

// NewMySQLShowCreateTableAction creates MySQL SHOW CREATE TABLE Action.
// This is an Action that can run `SHOW CREATE TABLE` command on MySQL service with given DSN.
func NewMySQLShowCreateTableAction(id string, params *agentpb.StartActionRequest_MySQLShowCreateTableParams) Action {
	return &mysqlShowCreateTableAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (a *mysqlShowCreateTableAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mysqlShowCreateTableAction) Type() string {
	return "mysql-show-create-table"
}

// Run runs an Action and returns output and error.
func (a *mysqlShowCreateTableAction) Run(ctx context.Context) ([]byte, error) {
	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).

	db, err := sql.Open("mysql", a.params.Dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	// use %#q to convert "table" to `"table"` and `table` to "`table`" to avoid SQL injections
	var tableName, tableDef string
	row := db.QueryRowContext(ctx, fmt.Sprintf("SHOW /* pmm-agent */ CREATE TABLE %#q", a.params.Table))
	if err = row.Scan(&tableName, &tableDef); err != nil {
		return nil, err
	}
	return []byte(tableDef), nil
}

func (a *mysqlShowCreateTableAction) sealed() {}
