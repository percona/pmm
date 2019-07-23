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

package parser

import (
	"fmt"
	"reflect"

	pgquery "github.com/lfittl/pg_query_go"
	pgquerynodes "github.com/lfittl/pg_query_go/nodes"
	"github.com/pkg/errors"
)

// ExtractTables extracts table names from query.
func ExtractTables(query string) (tables []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			// preserve stack
			err = errors.WithStack(fmt.Errorf("%v", r))
		}
	}()

	tree, err := pgquery.Parse(query)
	if err != nil {
		return nil, errors.Wrap(err, "error on parsing sql query")
	}
	tables = []string{}
	tableNames := make(map[string]bool)
	excludedtableNames := make(map[string]bool)
	for _, stmt := range tree.Statements {
		foundTables, excludeTables := extractTableNames(stmt)
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

	return
}

func extractTableNames(stmts ...pgquerynodes.Node) ([]string, []string) {
	var tables, excludeTables []string
	for _, stmt := range stmts {
		if isNilValue(stmt) {
			continue
		}
		var foundTables, tmpExcludeTables []string
		switch v := stmt.(type) {
		case pgquerynodes.RawStmt:
			return extractTableNames(v.Stmt)
		case pgquerynodes.SelectStmt: // Select queries
			foundTables, tmpExcludeTables = extractTableNames(v.FromClause, v.WhereClause, v.WithClause, v.Larg, v.Rarg)
		case pgquerynodes.InsertStmt: // Insert queries
			foundTables, tmpExcludeTables = extractTableNames(v.Relation, v.SelectStmt, v.WithClause)
		case pgquerynodes.UpdateStmt: // Update queries
			foundTables, tmpExcludeTables = extractTableNames(v.Relation, v.FromClause, v.WhereClause, v.WithClause)
		case pgquerynodes.DeleteStmt: // Delete queries
			foundTables, tmpExcludeTables = extractTableNames(v.Relation, v.WhereClause, v.WithClause)

		case pgquerynodes.JoinExpr: // Joins
			foundTables, tmpExcludeTables = extractTableNames(v.Larg, v.Rarg)

		case pgquerynodes.RangeVar: // Table name
			foundTables = []string{*v.Relname}

		case pgquerynodes.List:
			foundTables, tmpExcludeTables = extractTableNames(v.Items...)

		case pgquerynodes.WithClause: // To exclude temporary tables
			foundTables, tmpExcludeTables = extractTableNames(v.Ctes)
			for _, item := range v.Ctes.Items {
				if cte, ok := item.(pgquerynodes.CommonTableExpr); ok {
					tmpExcludeTables = append(tmpExcludeTables, *cte.Ctename)
				}
			}

		case pgquerynodes.A_Expr: // Where a=b
			foundTables, tmpExcludeTables = extractTableNames(v.Lexpr, v.Rexpr)

		// Subqueries
		case pgquerynodes.SubLink:
			foundTables, tmpExcludeTables = extractTableNames(v.Subselect, v.Xpr, v.Testexpr)
		case pgquerynodes.RangeSubselect:
			foundTables, tmpExcludeTables = extractTableNames(v.Subquery)
		case pgquerynodes.CommonTableExpr:
			foundTables, tmpExcludeTables = extractTableNames(v.Ctequery)

		default:
			if isPointer(v) { // to avoid duplications in case of pointers
				dereference, ok := reflect.ValueOf(v).Elem().Interface().(pgquerynodes.Node)
				if ok {
					foundTables, tmpExcludeTables = extractTableNames(dereference)
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
