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

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
)

type mysqlShowTableStatusAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLShowTableStatusParams
}

// NewMySQLShowTableStatusAction creates MySQL SHOW TABLE STATUS Action.
// This is an Action that can run `SHOW TABLE STATUS` command on MySQL service with given DSN.
func NewMySQLShowTableStatusAction(id string, params *agentpb.StartActionRequest_MySQLShowTableStatusParams) Action {
	return &mysqlShowTableStatusAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (a *mysqlShowTableStatusAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mysqlShowTableStatusAction) Type() string {
	return "mysql-show-table-status"
}

// Run runs an Action and returns output and error.
func (a *mysqlShowTableStatusAction) Run(ctx context.Context) ([]byte, error) {
	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).

	db, err := sql.Open("mysql", a.params.Dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, "SHOW /* pmm-agent */ TABLE STATUS WHERE Name = ?", a.params.Table)
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	if len(dataRows) == 0 {
		return nil, errors.Errorf("table %q not found", a.params.Table)
	}
	return jsonRows(columns, dataRows)
}

func (a *mysqlShowTableStatusAction) sealed() {}
