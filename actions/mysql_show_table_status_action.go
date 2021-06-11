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

	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"

	"github.com/percona/pmm-agent/tlshelpers"
)

type mysqlShowTableStatusAction struct {
	id     string
	params *agentpb.StartActionRequest_MySQLShowTableStatusParams
}

// NewMySQLShowTableStatusAction creates MySQL SHOW TABLE STATUS Action.
// This is an Action that can run `SHOW TABLE STATUS` command on MySQL service with given DSN.
func NewMySQLShowTableStatusAction(id string, params *agentpb.StartActionRequest_MySQLShowTableStatusParams) Action {
	return &mysqlShowTableStatusAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (a *mysqlShowTableStatusAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mysqlShowTableStatusAction) Type() string {
	return "mysql-show-table-status"
}

// Run runs an Action and returns output and error.
func (a *mysqlShowTableStatusAction) Run(ctx context.Context) ([]byte, error) {
	db, err := mysqlOpen(a.params.Dsn, a.params.TlsFiles)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck
	defer tlshelpers.DeregisterMySQLCerts()

	rows, err := db.QueryContext(ctx, "SHOW /* pmm-agent */ TABLE STATUS WHERE Name = ?", a.params.Table)
	if err != nil {
		return nil, err
	}

	columns, dataRows, err := readRows(rows)
	if err != nil {
		return nil, err
	}
	if len(dataRows) == 0 {
		return nil, errors.Errorf("table %q not found", a.params.Table)
	}
	return jsonRows(columns, dataRows)
}

func (a *mysqlShowTableStatusAction) sealed() {}
