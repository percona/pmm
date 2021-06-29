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
	"github.com/prometheus/common/log"

	"github.com/percona/pmm-agent/tlshelpers"
)

type mysqlExplainAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLExplainParams
	// query has a copy of the original params.Query field if the query is a SELECT or the equivalent
	// SELECT after converting DML queries.
	query      string
	isDMLQuery bool
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
func NewMySQLExplainAction(id string, params *agentpb.StartActionRequest_MySQLExplainParams) Action {
	if params.TlsFiles != nil && params.TlsFiles.Files != nil {
		err := tlshelpers.RegisterMySQLCerts(params.TlsFiles.Files)
		if err != nil {
			log.Error(err)
		}
	}

	ret := &mysqlExplainAction{
		id:         id,
		params:     params,
		query:      params.Query,
		isDMLQuery: isDMLQuery(params.Query),
	}

	if ret.isDMLQuery {
		ret.query = dmlToSelect(params.Query)
	}

	return ret
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
	db, err := mysqlOpen(a.params.Dsn, a.params.TlsFiles)
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
		Query:      a.query,
		IsDMLQuery: a.isDMLQuery,
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

func (a *mysqlExplainAction) explainDefault(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", a.query))
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
	err := tx.QueryRowContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ FORMAT=JSON %s", a.query)).Scan(&b)
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
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("EXPLAIN /* pmm-agent */ %s", a.query))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}
