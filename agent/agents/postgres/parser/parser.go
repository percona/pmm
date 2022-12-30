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

// Package parser contains functions for queries parsing.
package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v2"
	"github.com/pkg/errors"
)

var extractTablesRecover = true

// ExtractTables extracts table names from query.
func ExtractTables(query string) (tables []string, err error) {
	if extractTablesRecover {
		defer func() {
			if r := recover(); r != nil {
				// preserve stack
				err = errors.WithStack(fmt.Errorf("panic: %v", r))
			}
		}()
	}

	var jsonTree string
	if jsonTree, err = pgquery.ParseToJSON(query); err != nil {
		err = errors.Wrap(err, "error on parsing sql query")
		return
	}

	var treeMap map[string]json.RawMessage
	if err = json.Unmarshal([]byte(jsonTree), &treeMap); err != nil {
		err = errors.Wrap(err, "failed to unmarshal JSON")
		return
	}

	var stmts []json.RawMessage
	if err = json.Unmarshal([]byte(treeMap["stmts"]), &stmts); err != nil {
		err = errors.Wrap(err, "failed to unmarshal JSON")
		return
	}

	tableNames := make(map[string]bool)
	for _, stmt := range stmts {
		json := string(stmt)
		for _, v := range extract(json, `"relname":"`, `"`) {
			tableNames[v] = true
		}
		for _, v := range extract(json, `"ctename":"`, `"`) {
			delete(tableNames, v)
		}
	}

	tables = []string{}
	for k := range tableNames {
		tables = append(tables, k)
	}
	sort.Strings(tables)

	return
}

func extract(query, pre, post string) []string {
	re := regexp.MustCompile(fmt.Sprintf("(%s)(.*?)(%s)", pre, post))
	match := re.FindAll([]byte(query), -1)

	tables := []string{}
	for _, v := range match {
		tables = append(tables, parseValue(string(v), pre, post))
	}

	return tables
}

func parseValue(v, pre, post string) string {
	v = strings.Replace(v, pre, "", -1)
	return strings.Replace(v, post, "", -1)
}
