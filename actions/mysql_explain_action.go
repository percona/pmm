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
	"encoding/json"
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

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close() //nolint:errcheck

	switch e.params.OutputFormat {
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT:
		return e.explainDefault(ctx, conn)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON:
		return e.explainJSON(ctx, conn)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON:
		return e.explainTraditionalJSON(ctx, conn)
	default:
		return nil, errors.Errorf("unsupported output format %s", e.params.OutputFormat)
	}
}

func (e *mysqlExplainAction) sealed() {}

func (e *mysqlExplainAction) explainDefault(ctx context.Context, conn *sql.Conn) ([]byte, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", e.params.Query))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}

	// TODO Convert results to the output similar to mysql's CLI \G format
	// for compatibility with pt-visual-explain.
	// https://jira.percona.com/browse/PMM-4107

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	w.Write([]byte(strings.Join(columns, "\t"))) //nolint:errcheck
	for _, dataRow := range dataRows {
		row := "\n"
		for _, d := range dataRow {
			v := "NULL"
			if d != nil {
				v = fmt.Sprint(d)
			}
			row += v + "\t"
		}
		w.Write([]byte(row)) //nolint:errcheck
	}
	if err = w.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *mysqlExplainAction) explainJSON(ctx context.Context, conn *sql.Conn) ([]byte, error) {
	var b []byte
	err := conn.QueryRowContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ FORMAT=JSON %s", e.params.Query)).Scan(&b)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	// https://dev.mysql.com/doc/refman/8.0/en/explain-extended.html
	rows, err := conn.QueryContext(ctx, "SHOW /* pmm-agent */ WARNINGS")
	if err != nil {
		return b, nil // ingore error, return original output
	}
	defer rows.Close() //nolint:errcheck

	var warnings []map[string]interface{}
	for rows.Next() {
		var level, message string
		var code int
		if err = rows.Scan(&level, &code, &message); err != nil {
			continue
		}
		warnings = append(warnings, map[string]interface{}{
			"Level":   level,
			"Code":    code,
			"Message": message,
		})
	}
	// ignore rows.Err()

	m["warnings"] = warnings
	return json.Marshal(m)
}

func (e *mysqlExplainAction) explainTraditionalJSON(ctx context.Context, conn *sql.Conn) ([]byte, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", e.params.Query))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}
