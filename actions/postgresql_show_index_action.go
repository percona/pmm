// pmm-agent
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
