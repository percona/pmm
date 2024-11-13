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
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/sqlrows"
)

const postgreSQLQuerySelectActionType = "postgresql-query-select"

type postgresqlQuerySelectAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_PostgreSQLQuerySelectParams
	dsn     string
}

// NewPostgreSQLQuerySelectAction creates PostgreSQL SELECT query Action.
func NewPostgreSQLQuerySelectAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_PostgreSQLQuerySelectParams, tempDir string) (Action, error) {
	// A very basic check that there is a single SELECT query. It has oblivious false positives (`SELECT ';'`),
	// but PostgreSQL query lexical structure (https://www.postgresql.org/docs/current/sql-syntax-lexical.html)
	// does not allow false negatives.
	// If we decide to improve it, we could use our existing query parser from pg_stat_statement agent,
	// or use a simple hand-made parser similar to
	// https://github.com/mc2soft/pq-types/blob/ada769d4011a027a5385b9c4e47976fe327350a6/string_array.go#L82-L116
	if strings.Contains(params.Query, ";") {
		return nil, errors.New("query contains ';'")
	}

	dsn, err := templates.RenderDSN(params.Dsn, params.TlsFiles, filepath.Join(tempDir, postgreSQLQuerySelectActionType, id))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &postgresqlQuerySelectAction{
		id:      id,
		timeout: timeout,
		params:  params,
		dsn:     dsn,
	}, nil
}

// ID returns an Action ID.
func (a *postgresqlQuerySelectAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *postgresqlQuerySelectAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *postgresqlQuerySelectAction) Type() string {
	return postgreSQLQuerySelectActionType
}

// DSN returns the DSN for the Action.
func (a *postgresqlQuerySelectAction) DSN() string {
	return a.dsn
}

// Run runs an Action and returns output and error.
func (a *postgresqlQuerySelectAction) Run(ctx context.Context) ([]byte, error) {
	connector, err := pq.NewConnector(a.dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, "SELECT /* pmm-agent */ "+a.params.Query) //nolint:gosec
	if err != nil {
		return nil, errors.WithStack(err)
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return agentpb.MarshalActionQuerySQLResult(columns, dataRows)
}

func (a *postgresqlQuerySelectAction) sealed() {}
