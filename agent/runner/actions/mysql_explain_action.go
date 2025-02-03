// Copyright (C) 2023 Percona LLC
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
	"time"

	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/queryparser"
	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/sqlrows"
)

const (
	errNoDatabaseSelectedCode    = "Error 1046 (3D000)"
	errNoDatabaseSelectedMessage = "Database name is not included in this query. Explain could not be triggered without this info"
)

type mysqlExplainAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_MySQLExplainParams
}

type explainResponse struct {
	ExplainResult []byte `json:"explain_result"`
	Query         string `json:"explained_query"`
	IsDMLQuery    bool   `json:"is_dml"`
}

// ErrCannotEncodeExplainResponse cannot JSON encode the explain response.
var errCannotEncodeExplainResponse = errors.New("cannot JSON encode the explain response")

// NewMySQLExplainAction creates MySQL Explain Action.
// This is an Action that can run `EXPLAIN` command on MySQL service with given DSN.
func NewMySQLExplainAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_MySQLExplainParams) (Action, error) {
	if params.Query == "" {
		return nil, errors.New("Query to EXPLAIN is empty")
	}

	// You cant run Explain on trimmed queries.
	if strings.HasSuffix(params.Query, "...") {
		return nil, errors.New("EXPLAIN failed because the query exceeded max length and got trimmed. Set max-query-length to a larger value.") //nolint:revive
	}

	// Explain is supported only for DML queries.
	// https://dev.mysql.com/doc/refman/8.0/en/using-explain.html
	if !isDMLQuery(params.Query) {
		return nil, errors.New("EXPLAIN functionality is supported only for DML queries - SELECT, INSERT, UPDATE, DELETE and REPLACE.") //nolint:revive
	}

	return &mysqlExplainAction{
		id:      id,
		timeout: timeout,
		params:  params,
	}, nil
}

// ID returns an Action ID.
func (a *mysqlExplainAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *mysqlExplainAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *mysqlExplainAction) Type() string {
	return "mysql-explain"
}

// DSN returns a DSN for the Action.
func (a *mysqlExplainAction) DSN() string {
	return a.params.Dsn
}

// Run runs an Action and returns output and error.
func (a *mysqlExplainAction) Run(ctx context.Context) ([]byte, error) {
	a.params.Query = queryparser.GetMySQLFingerprintFromExplainFingerprint(a.params.Query)

	// query has a copy of the original params.Query field if the query is a SELECT or the equivalent
	// SELECT after converting DML queries.
	query, changedToSelect := dmlToSelect(a.params.Query)
	db, err := mysqlOpen(a.params.Dsn, a.params.TlsFiles, a.params.TlsSkipVerify)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck
	defer tlshelpers.DeregisterMySQLCerts()

	// Create a transaction to explain a query in to be able to rollback any
	// harm done by stored functions/procedures.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	response := explainResponse{
		Query:      query,
		IsDMLQuery: changedToSelect,
	}

	if a.params.Schema != "" {
		_, err = tx.ExecContext(ctx, fmt.Sprintf("USE %#q", a.params.Schema))
		if err != nil {
			return nil, err
		}
	}

	switch a.params.OutputFormat {
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_DEFAULT:
		response.ExplainResult, err = a.explainDefault(ctx, tx)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_JSON:
		response.ExplainResult, err = a.explainJSON(ctx, tx)
	case agentpb.MysqlExplainOutputFormat_MYSQL_EXPLAIN_OUTPUT_FORMAT_TRADITIONAL_JSON:
		response.ExplainResult, err = a.explainTraditionalJSON(ctx, tx)
	default:
		return nil, errors.Errorf("unsupported output format %s", a.params.OutputFormat)
	}

	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(response)
	if err != nil {
		return nil, errCannotEncodeExplainResponse
	}

	return b, nil
}

func (a *mysqlExplainAction) sealed() {}

func prepareValues(values []string) []any {
	res := make([]any, 0, len(values))
	for _, p := range values {
		res = append(res, p)
	}

	return res
}

func (a *mysqlExplainAction) explainDefault(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", a.params.Query), prepareValues(a.params.Values)...)
	if err != nil {
		if strings.Contains(err.Error(), errNoDatabaseSelectedCode) {
			return nil, errors.Wrap(err, errNoDatabaseSelectedMessage)
		}
		return nil, err
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
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
	err := tx.QueryRowContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ FORMAT=JSON %s", a.params.Query), prepareValues(a.params.Values)...).Scan(&b)
	if err != nil {
		if strings.Contains(err.Error(), errNoDatabaseSelectedCode) {
			return nil, errors.Wrap(err, errNoDatabaseSelectedMessage)
		}
		return nil, err
	}

	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	// https://dev.mysql.com/doc/refman/8.0/en/explain-extended.html
	rows, err := tx.QueryContext(ctx, "SHOW /* pmm-agent */ WARNINGS")
	if err != nil {
		// ignore error, return original output
		return b, nil //nolint:nilerr
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
	m["real_table_name"] = parseRealTableName(a.params.Query)

	return json.Marshal(m)
}

func (a *mysqlExplainAction) explainTraditionalJSON(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", a.params.Query), prepareValues(a.params.Values)...)
	if err != nil {
		if strings.Contains(err.Error(), errNoDatabaseSelectedCode) {
			return nil, errors.Wrap(err, errNoDatabaseSelectedMessage)
		}
		return nil, err
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}
