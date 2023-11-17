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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/utils/templates"
	agentpb "github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/sqlrows"
)

type postgresqlShowIndexAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_PostgreSQLShowIndexParams
	tempDir string
}

// NewPostgreSQLShowIndexAction creates PostgreSQL SHOW INDEX Action.
// This is an Action that can run `SHOW INDEX` command on PostgreSQL service with given DSN.
func NewPostgreSQLShowIndexAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_PostgreSQLShowIndexParams, tempDir string) Action {
	return &postgresqlShowIndexAction{
		id:      id,
		timeout: timeout,
		params:  params,
		tempDir: tempDir,
	}
}

// ID returns an Action ID.
func (a *postgresqlShowIndexAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *postgresqlShowIndexAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *postgresqlShowIndexAction) Type() string {
	return "postgresql-show-index"
}

// Run runs an Action and returns output and error.
func (a *postgresqlShowIndexAction) Run(ctx context.Context) ([]byte, error) {
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

	var namespaceQuery string
	var args []interface{}
	table := strings.Split(a.params.Table, ".")
	switch len(table) {
	case 2:
		args = append(args, table[1], table[0])
		namespaceQuery = "AND schemaname = $2"
	case 1:
		args = append(args, table[0])
	}
	// TODO: Throw error if table doesn't exist.
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT /* pmm-agent */ * FROM pg_indexes WHERE tablename = $1 %s", namespaceQuery), args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return jsonRows(columns, dataRows)
}

func (a *postgresqlShowIndexAction) sealed() {}
