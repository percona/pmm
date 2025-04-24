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
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/utils/templates"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/utils/sqlrows"
)

const postgreSQLShowIndexActionType = "postgresql-show-index"

type postgresqlShowIndexAction struct {
	id      string
	timeout time.Duration
	params  *agentv1.StartActionRequest_PostgreSQLShowIndexParams
	dsn     string
	tmpDir  string
}

// NewPostgreSQLShowIndexAction creates PostgreSQL SHOW INDEX Action.
// This is an Action that can run `SHOW INDEX` command on PostgreSQL service with given DSN.
func NewPostgreSQLShowIndexAction(id string, timeout time.Duration, params *agentv1.StartActionRequest_PostgreSQLShowIndexParams, tempDir string) (Action, error) {
	tmpDir := filepath.Join(tempDir, postgreSQLShowIndexActionType, id)
	dsn, err := templates.RenderDSN(params.Dsn, params.TlsFiles, tmpDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &postgresqlShowIndexAction{
		id:      id,
		timeout: timeout,
		params:  params,
		dsn:     dsn,
		tmpDir:  tmpDir,
	}, nil
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
	return postgreSQLShowIndexActionType
}

// DSN returns a DSN for the Action.
func (a *postgresqlShowIndexAction) DSN() string {
	return a.dsn
}

// Run runs an Action and returns output and error.
func (a *postgresqlShowIndexAction) Run(ctx context.Context) ([]byte, error) {
	defer templates.CleanupTempDir(a.tmpDir, logrus.WithField("component", postgreSQLShowIndexActionType))

	connector, err := pq.NewConnector(a.dsn)
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
