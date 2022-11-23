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
	"time"

	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/api/agentpb"
)

type postgresqlExplainAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_PostgreSQLExplainParams
	query   string
}

// NewPostgreSQLExplainAction creates PostgreSQL Explain Action.
// This is an Action that can run `EXPLAIN` command on PostgreSQL service with given DSN.
func NewPostgreSQLExplainAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_PostgreSQLExplainParams) Action {
	return &postgresqlExplainAction{
		id:      id,
		timeout: timeout,
		params:  params,
		query:   params.Query,
	}
}

// ID returns an Action ID.
func (a *postgresqlExplainAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *postgresqlExplainAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *postgresqlExplainAction) Type() string {
	return "mysql-explain"
}

// Run runs an Action and returns output and error.
func (a *postgresqlExplainAction) Run(ctx context.Context) ([]byte, error) {
	// query has a copy of the original params.Query field if the query is a SELECT or the equivalent
	// SELECT after converting DML queries.
	query := a.query
	isDMLQuery := isDMLQuery(query)
	if isDMLQuery {
		query = dmlToSelect(query)
	}
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
		Query:      query,
		IsDMLQuery: isDMLQuery,
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

func (a *postgresqlExplainAction) sealed() {}

func (a *postgresqlExplainAction) explainDefault(ctx context.Context, tx *sql.Tx) ([]byte, error) {
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
