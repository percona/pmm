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

// Package actions provides Actions implementations.
package actions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/tlshelpers"
	"github.com/percona/pmm/api/agentpb"
)

const queryTag = "pmm-agent-tests:MySQLVersion"

var whiteSpacesRegExp = regexp.MustCompile(`\s+`)

// jsonRows converts input to JSON array:
// [
//
//	["column 1", "column 2", …],
//	["value 1", 2, …]
//	…
//
// ].
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
func mysqlOpen(dsn string, tlsFiles *agentpb.TextFiles, tlsSkipVerify bool) (*sql.DB, error) {
	if tlsFiles != nil {
		err := tlshelpers.RegisterMySQLCerts(tlsFiles.Files, tlsSkipVerify)
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

func prepareRealTableName(name string) string {
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "`", "")
	return strings.TrimSpace(name)
}

func parseRealTableName(query string) string {
	query = whiteSpacesRegExp.ReplaceAllString(query, " ")
	// due to historical reasons we parsing only one table name
	keyword := "FROM "

	query = strings.ReplaceAll(query, " . ", ".")
	// in case of subquery it will choose root query
	index := strings.LastIndex(query, keyword)
	if index == -1 {
		return ""
	}

	parsed := query[index+len(keyword):]
	parsed = strings.ReplaceAll(parsed, ";", "")
	index = strings.Index(parsed, " ")
	if index == -1 {
		return strings.TrimSpace(parsed)
	}

	return strings.TrimSpace(parsed[:index+1])
}

func prepareQueryWithDatabaseTableName(query, name string) string {
	// use %#q to convert "table" to `"table"` and `table` to "`table`" to avoid SQL injections
	q := fmt.Sprintf("%s %#q", query, prepareRealTableName(name))
	if !strings.Contains(q, ".") {
		return q
	}

	// handle case when there is table name together with database name
	return strings.ReplaceAll(q, ".", "`.`")
}
