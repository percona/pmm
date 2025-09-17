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

// Package parser contains functions for queries parsing.
package parser

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
	"github.com/pkg/errors"
)

var extractTablesRecover = true

// ExtractTables extracts table names from query.
func ExtractTables(query string) ([]string, error) {
	var err error
	var tables []string //nolint:prealloc

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
		return nil, err
	}

	var res []string
	tableNames := make(map[string]struct{})
	res, err = extract(jsonTree, `"relname":"`, `"`)
	if err != nil {
		return nil, err
	}
	for _, v := range res {
		tableNames[v] = struct{}{}
	}
	res, err = extract(jsonTree, `"ctename":"`, `"`)
	if err != nil {
		return nil, err
	}
	for _, v := range res {
		delete(tableNames, v)
	}

	for k := range tableNames {
		tables = append(tables, k)
	}
	sort.Strings(tables)

	return tables, nil
}

func extract(query, pre, post string) ([]string, error) {
	re, err := regexp.Compile(fmt.Sprintf("(%s)(.*?)(%s)", pre, post))
	if err != nil {
		return nil, err
	}

	match := re.FindAll([]byte(query), -1)
	tables := make([]string, 0, len(match))
	for _, v := range match {
		tables = append(tables, parseValue(string(v), pre, post))
	}

	return tables, nil
}

func parseValue(v, pre, post string) string {
	v = strings.ReplaceAll(v, pre, "")
	return strings.ReplaceAll(v, post, "")
}
