package encryption

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

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

func (c DatabaseConnection) DSN() string {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}

	if c.Password != "" {
		c.Password = fmt.Sprintf("password=%s", c.Password)
	}

	return fmt.Sprintf("host=%s port=%d user=%s %s sslmode=%s", c.Host, c.Port, c.User, c.Password, c.SSLMode)
}

func (item EncryptedItem) Read(tx *sql.Tx) (*QueryValues, error) {
	what := append(item.Identificators, item.Columns...)
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(what, ","), item.Table)
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
			set = append(set, fmt.Sprintf("%s = $%d", item.Columns[k], i))
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
	err = rows.Close()
	if err != nil {
		return nil, err
	}

	return q, nil
}
