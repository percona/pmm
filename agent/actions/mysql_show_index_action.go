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
	"fmt"

	"github.com/percona/pmm/api/agentpb"

	"github.com/percona/pmm-agent/tlshelpers"
)

type mysqlShowIndexAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLShowIndexParams
}

// NewMySQLShowIndexAction creates MySQL SHOW INDEX Action.
// This is an Action that can run `SHOW INDEX` command on MySQL service with given DSN.
func NewMySQLShowIndexAction(id string, params *agentpb.StartActionRequest_MySQLShowIndexParams) Action {
	return &mysqlShowIndexAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (a *mysqlShowIndexAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mysqlShowIndexAction) Type() string {
	return "mysql-show-index"
}

// Run runs an Action and returns output and error.
func (a *mysqlShowIndexAction) Run(ctx context.Context) ([]byte, error) {
	db, err := mysqlOpen(a.params.Dsn, a.params.TlsFiles)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck
	defer tlshelpers.DeregisterMySQLCerts()

	// use %#q to convert "table" to `"table"` and `table` to "`table`" to avoid SQL injections
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SHOW /* pmm-agent */ INDEX IN %#q", a.params.Table))
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonRows(columns, dataRows)
}

func (a *mysqlShowIndexAction) sealed() {}
