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
	"time"

	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/utils/sqlrows"
)

type mysqlShowIndexAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_MySQLShowIndexParams
}

// NewMySQLShowIndexAction creates MySQL SHOW INDEX Action.
// This is an Action that can run `SHOW INDEX` command on MySQL service with given DSN.
func NewMySQLShowIndexAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_MySQLShowIndexParams) Action {
	return &mysqlShowIndexAction{
		id:      id,
		timeout: timeout,
		params:  params,
	}
}

// ID returns an Action ID.
func (a *mysqlShowIndexAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *mysqlShowIndexAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *mysqlShowIndexAction) Type() string {
	return "mysql-show-index"
}

// DSN returns a DSN for the Action.
func (a *mysqlShowIndexAction) DSN() string {
	return a.params.Dsn
}

// Run runs an Action and returns output and error.
func (a *mysqlShowIndexAction) Run(ctx context.Context) ([]byte, error) {
	db, err := mysqlOpen(a.params.Dsn, a.params.TlsFiles, a.params.TlsSkipVerify)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck
	defer tlshelpers.DeregisterMySQLCerts()

	rows, err := db.QueryContext(ctx, prepareQueryWithDatabaseTableName("SHOW /* pmm-agent */ INDEX IN", a.params.Table))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}

func (a *mysqlShowIndexAction) sealed() {}
