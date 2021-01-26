// pmm-agent
// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actions

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

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
func (a *mysqlExplainAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mysqlExplainAction) Type() string {
	return "mysql-explain"
}

// Run runs an Action and returns output and error.
func (a *mysqlExplainAction) Run(ctx context.Context) ([]byte, error) {
	db, err := mysqlOpen(a.params.Dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	// Create a transaction to explain a query in to be able to rollback any
	// harm done by stored functions/procedures.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	switch a.params.OutputFormat {
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT:
		return a.explainDefault(ctx, tx)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON:
		return a.explainJSON(ctx, tx)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON:
		return a.explainTraditionalJSON(ctx, tx)
	default:
		return nil, errors.Errorf("unsupported output format %s", a.params.OutputFormat)
	}
}

func (a *mysqlExplainAction) sealed() {}

func (a *mysqlExplainAction) explainDefault(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", a.params.Query))
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

func (a *mysqlExplainAction) explainJSON(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var b []byte
	err := tx.QueryRowContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ FORMAT=JSON %s", a.params.Query)).Scan(&b)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	// https://dev.mysql.com/doc/refman/8.0/en/explain-extended.html
	rows, err := tx.QueryContext(ctx, "SHOW /* pmm-agent */ WARNINGS")
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

func (a *mysqlExplainAction) explainTraditionalJSON(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", a.params.Query))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}
