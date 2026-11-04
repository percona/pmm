// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package sqlrows provides helper methods for *sql.Rows.
package sqlrows

import "database/sql"

// ReadRows reads and closes given *sql.Rows, returning columns, data rows, and first encountered error.
func ReadRows(rows *sql.Rows) ([]string, [][]any, error) {
	var columns []string
	var dataRows [][]any
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
		dest := make([]any, len(columns))
		for i := range dest {
			var ei any
			dest[i] = &ei
		}
		err = rows.Scan(dest...)
		if err != nil {
			return columns, dataRows, err
		}

		// Each dest element is an *interface{} (&ei above) which always contain some typed data
		// (in particular, it can contain typed nil). Dereference it for easier manipulations by the caller.
		// As a special case, convert []byte values to strings. That does not change semantics of this function
		// (Go string can contain any byte sequence), but prevents json.Marshal (at jsonRows) from encoding
		// them as base64 strings.
		for i, d := range dest {
			ei := *d.(*any) //nolint:forcetypeassert
			dest[i] = ei
			if b, ok := ei.([]byte); ok {
				dest[i] = string(b)
			}
		}

		dataRows = append(dataRows, dest)
	}
	err = rows.Err()

	return columns, dataRows, err
}
