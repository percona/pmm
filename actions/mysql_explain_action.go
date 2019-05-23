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
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/tabwriter"

	_ "github.com/go-sql-driver/mysql" // register SQL driver
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
)

type mysqlExplainAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLExplainParams
}

// NewMySQLExplainAction creates MySQL Explain Action.
// This is an Action that can run `EXPLAIN` command on MySQL service with given DSN.
func NewMySQLExplainAction(id string, params *agentpb.StartActionRequest_MySQLExplainParams) Action {
	return &mysqlExplainAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (e *mysqlExplainAction) ID() string {
	return e.id
}

// Type returns an Action type.
func (e *mysqlExplainAction) Type() string {
	return "mysql-explain"
}

// Run runs an Action and returns output and error.
func (e *mysqlExplainAction) Run(ctx context.Context) ([]byte, error) {
	// TODO Use sql.OpenDB with ctx when https://github.com/go-sql-driver/mysql/issues/671 is released
	// (likely in version 1.5.0).

	db, err := sql.Open("mysql", e.params.Dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	switch e.params.OutputFormat {
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT:
		return e.explainDefault(ctx, db)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON:
		return e.explainJSON(ctx, db)
	default:
		return nil, errors.Errorf("unsupported output format %s", e.params.OutputFormat)
	}
}

func (e *mysqlExplainAction) sealed() {}

func (e *mysqlExplainAction) explainDefault(ctx context.Context, db *sql.DB) ([]byte, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", e.params.Query))
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	w.Write([]byte(strings.Join(columns, "\t"))) //nolint:errcheck
	for rows.Next() {
		dest := make([]interface{}, len(columns))
		for i := range dest {
			var sp *string
			dest[i] = &sp
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, err
		}

		row := "\n"
		for _, d := range dest {
			v := "NULL"
			if sp := *d.(**string); sp != nil {
				v = *sp
			}
			row += v + "\t"
		}
		w.Write([]byte(row)) //nolint:errcheck
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if err = w.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *mysqlExplainAction) explainJSON(ctx context.Context, db *sql.DB) ([]byte, error) {
	var res string
	err := db.QueryRowContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ FORMAT=JSON %s", e.params.Query)).Scan(&res)
	return []byte(res), err
}
