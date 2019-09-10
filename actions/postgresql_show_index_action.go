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
	"context"
	"database/sql"

	"github.com/lib/pq"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
)

type postgresqlShowIndexAction struct {
	id     string
	params *agentpb.StartActionRequest_PostgreSQLShowIndexParams
}

// NewPostgreSQLShowIndexAction creates PostgreSQL SHOW INDEX Action.
// This is an Action that can run `SHOW INDEX` command on PostgreSQL service with given DSN.
func NewPostgreSQLShowIndexAction(id string, params *agentpb.StartActionRequest_PostgreSQLShowIndexParams) Action {
	return &postgresqlShowIndexAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (a *postgresqlShowIndexAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *postgresqlShowIndexAction) Type() string {
	return "postgresql-show-index"
}

// Run runs an Action and returns output and error.
func (a *postgresqlShowIndexAction) Run(ctx context.Context) ([]byte, error) {
	connector, err := pq.NewConnector(a.params.Dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	db := sql.OpenDB(connector)
	defer db.Close() //nolint:errcheck

	// TODO: Throw error if table doesn't exist.
	rows, err := db.QueryContext(ctx, "SELECT /* pmm-agent */ * FROM pg_indexes WHERE tablename = $1", a.params.Table)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return jsonRows(columns, dataRows)
}

func (a *postgresqlShowIndexAction) sealed() {}
