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

// Package actions provides Actions implementations and runner.
package actions

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Action describes an abstract thing that can be run by a client and return some output.
type Action interface {
	// ID returns an Action ID.
	ID() string
	// Type returns an Action type.
	Type() string
	// Run runs an Action and returns output and error.
	Run(ctx context.Context) ([]byte, error)

	sealed()
}

// readRows reads and closes given *sql.Rows, returning columns, data rows, and first encountered error.
func readRows(rows *sql.Rows) (columns []string, dataRows [][]interface{}, err error) {
	defer func() {
		// overwrite err with e only if err does not already contains (more interesting) error
		if e := rows.Close(); err == nil {
			err = e
		}
	}()

	columns, err = rows.Columns()
	if err != nil {
		return
	}

	for rows.Next() {
		dest := make([]interface{}, len(columns))
		for i := range dest {
			var ei interface{}
			dest[i] = &ei
		}
		if err = rows.Scan(dest...); err != nil {
			return
		}

		// Each dest element is an *interface{} (&ei above) which always contain some typed data
		// (in particular, it can contain typed nil). Dereference it for easier manipulations by the caller.
		// As a special case, convert []byte values to strings. That does not change semantics of this function,
		// but prevents json.Marshal (at jsonRows) from encoding them as base64 strings.
		for i, d := range dest {
			ei := *(d.(*interface{}))
			dest[i] = ei
			if b, ok := (ei).([]byte); ok {
				dest[i] = string(b)
			}
		}

		dataRows = append(dataRows, dest)
	}
	err = rows.Err()
	return //nolint:nakedret
}

// jsonRows converts input to JSON array:
// [
//   ["column 1", "columnt 2", …],
//   ["value 1", 2, …]
//   …
// ]
func jsonRows(columns []string, dataRows [][]interface{}) ([]byte, error) {
	res := make([][]interface{}, len(dataRows)+1)

	res[0] = make([]interface{}, len(columns))
	for i, col := range columns {
		res[0][i] = col
	}

	for i, row := range dataRows {
		res[i+1] = make([]interface{}, len(columns))
		copy(res[i+1], row)
	}

	return json.Marshal(res)
}
