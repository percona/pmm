// Copyright (C) 2024 Percona LLC
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

// Package sqlrows provides helper methods for *sql.Rows.
package sqlrows

import "database/sql"

// ReadRows reads and closes given *sql.Rows, returning columns, data rows, and first encountered error.
func ReadRows(rows *sql.Rows) ([]string, [][]interface{}, error) {
	var columns []string
	var dataRows [][]interface{}
	var err error

	defer func() {
		// overwrite err with e only if err does not already contain (a more interesting) error
		if e := rows.Close(); err == nil {
			err = e
		}
	}()

	columns, err = rows.Columns()
	if err != nil {
		return columns, dataRows, err
	}

	for rows.Next() {
		dest := make([]interface{}, len(columns))
		for i := range dest {
			var ei interface{}
			dest[i] = &ei
		}
		if err = rows.Scan(dest...); err != nil {
			return columns, dataRows, err
		}

		// Each dest element is an *interface{} (&ei above) which always contain some typed data
		// (in particular, it can contain typed nil). Dereference it for easier manipulations by the caller.
		// As a special case, convert []byte values to strings. That does not change semantics of this function
		// (Go string can contain any byte sequence), but prevents json.Marshal (at jsonRows) from encoding
		// them as base64 strings.
		for i, d := range dest {
			ei := *(d.(*interface{})) //nolint:forcetypeassert
			dest[i] = ei
			if b, ok := (ei).([]byte); ok {
				dest[i] = string(b)
			}
		}

		dataRows = append(dataRows, dest)
	}
	err = rows.Err()

	return columns, dataRows, err
}
