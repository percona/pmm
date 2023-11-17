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
	agentpb "github.com/percona/pmm/api/agentpb/v1"
	"github.com/percona/pmm/utils/sqlrows"
)

type postgresqlQueryShowAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_PostgreSQLQueryShowParams
	tempDir string
}

// NewPostgreSQLQueryShowAction creates PostgreSQL SHOW query Action.
func NewPostgreSQLQueryShowAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_PostgreSQLQueryShowParams, tempDir string) Action {
	return &postgresqlQueryShowAction{
		id:      id,
		timeout: timeout,
		params:  params,
		tempDir: tempDir,
	}
}

// ID returns an Action ID.
func (a *postgresqlQueryShowAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *postgresqlQueryShowAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *postgresqlQueryShowAction) Type() string {
	return "postgresql-query-show"
}

// Run runs an Action and returns output and error.
func (a *postgresqlQueryShowAction) Run(ctx context.Context) ([]byte, error) {
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

	rows, err := db.QueryContext(ctx, "SHOW /* pmm-agent */ ALL")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return agentpb.MarshalActionQuerySQLResult(columns, dataRows)
}

func (a *postgresqlQueryShowAction) sealed() {}
