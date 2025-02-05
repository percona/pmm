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
)

type mysqlShowCreateTableAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_MySQLShowCreateTableParams
}

// NewMySQLShowCreateTableAction creates MySQL SHOW CREATE TABLE Action.
// This is an Action that can run `SHOW CREATE TABLE` command on MySQL service with given DSN.
func NewMySQLShowCreateTableAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_MySQLShowCreateTableParams) Action {
	return &mysqlShowCreateTableAction{
		id:      id,
		timeout: timeout,
		params:  params,
	}
}

// ID returns an Action ID.
func (a *mysqlShowCreateTableAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *mysqlShowCreateTableAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *mysqlShowCreateTableAction) Type() string {
	return "mysql-show-create-table"
}

// DSN returns a DSN for the Action.
func (a *mysqlShowCreateTableAction) DSN() string {
	return a.params.Dsn
}

// Run runs an Action and returns output and error.
func (a *mysqlShowCreateTableAction) Run(ctx context.Context) ([]byte, error) {
	db, err := mysqlOpen(a.params.Dsn, a.params.TlsFiles, a.params.TlsSkipVerify)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck
	defer tlshelpers.DeregisterMySQLCerts()

	var tableName, tableDef string
	row := db.QueryRowContext(ctx, prepareQueryWithDatabaseTableName("SHOW /* pmm-agent */ CREATE TABLE", a.params.Table))
	if err = row.Scan(&tableName, &tableDef); err != nil {
		return nil, err
	}
	return []byte(tableDef), nil
}

func (a *mysqlShowCreateTableAction) sealed() {}
