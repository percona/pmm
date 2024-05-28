package encryption

import (
	"database/sql"
	"fmt"
)

func prepareRowPointers(rows *sql.Rows) ([]any, error) {
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	columns := make(map[string]string)
	for _, columnType := range columnTypes {
		columns[columnType.Name()] = columnType.DatabaseTypeName()
	}

	row := []any{}
	for _, t := range columns {
		switch t {
		case "VARCHAR":
			row = append(row, new(string))
		default:
			// TODO support more identificators types
			return nil, fmt.Errorf("unsupported identificator type %s", t)
		}
	}

	return row, nil
}
