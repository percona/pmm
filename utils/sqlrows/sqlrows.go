package sqlrows

import "database/sql"

// ReadRows reads and closes given *sql.Rows, returning columns, data rows, and first encountered error.
func ReadRows(rows *sql.Rows) (columns []string, dataRows [][]interface{}, err error) {
	defer func() {
		// overwrite err with e only if err does not already contain (a more interesting) error
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
		// As a special case, convert []byte values to strings. That does not change semantics of this function
		// (Go string can contain any byte sequence), but prevents json.Marshal (at jsonRows) from encoding
		// them as base64 strings.
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
