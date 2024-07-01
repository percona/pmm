// Copyright (C) 2023 Percona LLC
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

package encryption

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"

	_ "github.com/lib/pq" // register SQL driver
)

// Connect open connection to DB.
func (c DatabaseConnection) Connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.DSN())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// DSN returns formatted connection string to PG.
func (c DatabaseConnection) DSN() string {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}

	if c.Password != "" {
		c.Password = fmt.Sprintf("password=%s", c.Password)
	}

	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s %s sslmode=%s sslrootcert=%s sslkey=%s sslcert=%s",
		c.Host, c.Port, c.DBName, c.User, c.Password, c.SSLMode, c.SSLCAPath, c.SSLKeyPath, c.SSLCertPath,
	)
}

func (item Table) ColumnsList() []string {
	res := []string{}
	for _, c := range item.Columns {
		res = append(res, c.Column)
	}

	return res
}

// Read returns query and it's values based on input.
func (item Table) Read(tx *sql.Tx) (*QueryValues, error) {
	what := slices.Concat(item.Identificators, item.ColumnsList())
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(what, ", "), item.Table) //nolint:gosec
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}

	q := new(QueryValues)
	for rows.Next() {
		row, err := prepareRowPointers(rows)
		if err != nil {
			return nil, err
		}
		err = rows.Scan(row...)
		if err != nil {
			return nil, err
		}

		i := 1
		set := []string{}
		setValues := []any{}
		for k, v := range row[len(item.Identificators):] {
			set = append(set, fmt.Sprintf("%s = $%d", item.Columns[k].Column, i))
			setValues = append(setValues, v)
			i++
		}
		setSQL := fmt.Sprintf("SET %s", strings.Join(set, ", "))
		q.SetValues = append(q.SetValues, setValues)

		where := []string{}
		whereValues := []any{}
		for k, id := range item.Identificators {
			where = append(where, fmt.Sprintf("%s = $%d", id, i))
			whereValues = append(whereValues, row[k])
			i++
		}
		whereSQL := fmt.Sprintf("WHERE %s", strings.Join(where, " AND "))
		q.WhereValues = append(q.WhereValues, whereValues)

		q.Query = fmt.Sprintf("UPDATE %s %s %s", item.Table, setSQL, whereSQL)
	}
	err = rows.Close() //nolint:sqlclosecheck
	if err != nil {
		return nil, err
	}

	return q, nil
}
