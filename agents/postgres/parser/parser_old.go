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

package parser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	pgquery "github.com/lfittl/pg_query_go"
	pgquerynodes "github.com/lfittl/pg_query_go/nodes"
	"github.com/pkg/errors"
)

// extractTablesOld extracts table names from query.
func extractTablesOld(query string) (tables []string, err error) {
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

	var tree pgquery.ParsetreeList
	if err = json.Unmarshal([]byte(jsonTree), &tree); err != nil {
		err = errors.Wrap(err, "failed to unmarshal JSON")
		return
	}

	tables = []string{}
	tableNames := make(map[string]bool)
	excludedtableNames := make(map[string]bool)
	for _, stmt := range tree.Statements {
		foundTables, excludeTables := extractTableNamesOld(stmt)
		for _, tableName := range excludeTables {
			if _, ok := excludedtableNames[tableName]; !ok {
				excludedtableNames[tableName] = true
			}
		}
		for _, tableName := range foundTables {
			_, tableAdded := tableNames[tableName]
			_, tableExcluded := excludedtableNames[tableName]
			if !tableAdded && !tableExcluded {
				tables = append(tables, tableName)
				tableNames[tableName] = true
			}
		}
	}

	sort.Strings(tables)

	return
}

func extractTableNamesOld(stmts ...pgquerynodes.Node) ([]string, []string) {
	var tables, excludeTables []string
	for _, stmt := range stmts {
		if isNilValue(stmt) {
			continue
		}
		var foundTables, tmpExcludeTables []string
		switch v := stmt.(type) {
		case pgquerynodes.RawStmt:
			return extractTableNamesOld(v.Stmt)
		case pgquerynodes.SelectStmt: // Select queries
			foundTables, tmpExcludeTables = extractTableNamesOld(v.FromClause, v.WhereClause, v.WithClause, v.Larg, v.Rarg)
		case pgquerynodes.InsertStmt: // Insert queries
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Relation, v.SelectStmt, v.WithClause)
		case pgquerynodes.UpdateStmt: // Update queries
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Relation, v.FromClause, v.WhereClause, v.WithClause)
		case pgquerynodes.DeleteStmt: // Delete queries
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Relation, v.WhereClause, v.WithClause)

		case pgquerynodes.JoinExpr: // Joins
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Larg, v.Rarg)

		case pgquerynodes.RangeVar: // Table name
			foundTables = []string{*v.Relname}

		case pgquerynodes.List:
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Items...)

		case pgquerynodes.WithClause: // To exclude temporary tables
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Ctes)
			for _, item := range v.Ctes.Items {
				if cte, ok := item.(pgquerynodes.CommonTableExpr); ok {
					tmpExcludeTables = append(tmpExcludeTables, *cte.Ctename)
				}
			}

		case pgquerynodes.A_Expr: // Where a=b
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Lexpr, v.Rexpr)

		// Subqueries
		case pgquerynodes.SubLink:
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Subselect, v.Xpr, v.Testexpr)
		case pgquerynodes.RangeSubselect:
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Subquery)
		case pgquerynodes.CommonTableExpr:
			foundTables, tmpExcludeTables = extractTableNamesOld(v.Ctequery)

		default:
			if isPointer(v) { // to avoid duplications in case of pointers
				dereference, ok := reflect.ValueOf(v).Elem().Interface().(pgquerynodes.Node)
				if ok {
					foundTables, tmpExcludeTables = extractTableNamesOld(dereference)
				}
			}
		}
		tables = append(tables, foundTables...)
		excludeTables = append(excludeTables, tmpExcludeTables...)
	}

	return tables, excludeTables
}

func isNilValue(i interface{}) bool {
	return i == nil || (isPointer(i) && reflect.ValueOf(i).IsNil())
}

func isPointer(v interface{}) bool {
	return reflect.ValueOf(v).Kind() == reflect.Ptr
}
