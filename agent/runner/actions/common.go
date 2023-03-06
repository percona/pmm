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

// Package actions provides Actions implementations.
package actions

import (
	"database/sql"
	"encoding/json"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/api/agentpb"
)

const queryTag = "pmm-agent-tests:MySQLVersion"

// jsonRows converts input to JSON array:
// [
//
//	["column 1", "column 2", …],
//	["value 1", 2, …]
//	…
//
// ]
func jsonRows(columns []string, dataRows [][]interface{}) ([]byte, error) {
	res := make([][]interface{}, len(dataRows)+1)

	res[0] = make([]interface{}, len(columns))
	for i, col := range columns {
		res[0][i] = col
	}

	for i, row := range dataRows {
		res[i+1] = make([]interface{}, len(columns))
		copy(res[i+1], row)
	}

	return json.Marshal(res)
}

// mysqlOpen returns *sql.DB for given MySQL DSN.
func mysqlOpen(dsn string, tlsFiles *agentpb.TextFiles) (*sql.DB, error) {
	if tlsFiles != nil {
		err := tlshelpers.RegisterMySQLCerts(tlsFiles.Files)
		if err != nil {
			return nil, err
		}
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return sql.OpenDB(connector), nil
}
