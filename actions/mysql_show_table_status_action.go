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
	"encoding/json"

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
func (e *mysqlShowTableStatusAction) ID() string {
	return e.id
}

// Type returns an Action type.
func (e *mysqlShowTableStatusAction) Type() string {
	return "mysql-table-status"
}

// Run runs an Action and returns output and error.
func (e *mysqlShowTableStatusAction) Run(ctx context.Context) ([]byte, error) {
	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).

	db, err := sql.Open("mysql", e.params.Dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, "SHOW /* pmm-agent */ TABLE STATUS WHERE Name = ?", e.params.Table)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		if rows.Err() == nil {
			return nil, errors.Errorf("table %q not found", e.params.Table)
		}
		return nil, errors.Errorf("failed to get first row: %v", rows.Err())
	}

	dest := make([]interface{}, len(columns))
	for i := range dest {
		var ei interface{}
		dest[i] = &ei
	}
	if err = rows.Scan(dest...); err != nil {
		return nil, err
	}

	if rows.Next() {
		return nil, errors.Errorf("unexpected second row")
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Each dest element is an *interface{} (&ei above) which can be nil for NULL values, or contain some typed data.
	// Convert []byte to string to prevent json.Marshal from encode it as base64 string.
	for i, d := range dest {
		if eip, ok := d.(*interface{}); ok && eip != nil {
			if b, ok := (*eip).([]byte); ok {
				dest[i] = string(b)
			}
		}
	}

	res := make(map[string]interface{}, len(columns))
	for i, col := range columns {
		res[col] = dest[i]
	}
	return json.Marshal(res)
}

func (e *mysqlShowTableStatusAction) sealed() {}
