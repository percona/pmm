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
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
)

type postgresqlExplainAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_PostgreSQLExplainParams
	query   string
	tempDir string
}

// NewPostgreSQLExplainAction creates PostgreSQL Explain Action.
// This is an Action that can run `EXPLAIN` command on PostgreSQL service with given DSN.
func NewPostgreSQLExplainAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_PostgreSQLExplainParams, tempDir string) Action {
	return &postgresqlExplainAction{
		id:      id,
		timeout: timeout,
		params:  params,
		query:   params.Query,
		tempDir: tempDir,
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
	dsn, err := templates.RenderDSN(a.params.Dsn, a.params.TlsFiles, filepath.Join(a.tempDir, strings.ToLower(a.Type()), a.id))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	connector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	response := explainResponse{
		// Query:      query,
		// IsDMLQuery: isDMLQuery,
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
	return nil, nil
}
