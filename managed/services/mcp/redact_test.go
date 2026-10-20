// Copyright (C) 2023 Percona LLC
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

package mcp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSQLLiterals(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		in           string
		doubleQuoted bool
		want         string
	}{
		{"MySQLString", "(`shop`.`customers`.`email` = 'user42@example.com')", true, "(`shop`.`customers`.`email` = ?)"},
		{"Numbers", "(`t`.`amount` > 450.00) and (`t`.`id` in (1,2,3))", true, "(`t`.`amount` > ?) and (`t`.`id` in (?,?,?))"},
		{"DigitsInsideIdentifiers", "`sbtest1`.`k` = 42 and col1 = 5", true, "`sbtest1`.`k` = ? and col1 = ?"},
		{"EscapedQuotes", "a = 'it''s' or b = 'a\\'b'", true, "a = ? or b = ?"},
		{"Hex", "x = 0x1F", true, "x = ?"},
		{"Exponent", "v > 1.5e10", true, "v > ?"},
		{"MySQLDoubleQuotedString", `x = "abc"`, true, "x = ?"},
		{"Unterminated", "x = 'abc", true, "x = ?"},
		{"PostgreSQLQuotedIdentifier", `("Email")::text = 'a@b'::text`, false, `("Email")::text = ?::text`},
		{"PostgreSQLPlaceholder", "id = $1", false, "id = $1"},
		// A backslash ends a PostgreSQL string (standard_conforming_strings) and
		// a MySQL one under NO_BACKSLASH_ESCAPES. Read as an escape, it would
		// pair the quotes wrongly and print 'hunter2' in clear; the text is
		// over-masked instead.
		{"PostgreSQLTrailingBackslash", `((path)::text = 'C:\'::text) AND ((secret)::text = 'hunter2'::text)`, false, "((path)::text = ?"},
		{"MySQLNoBackslashEscapes", `path = 'C:\' and pw = 'hunter2'`, true, "path = ?"},
		{"PostgreSQLDollarQuoted", "SELECT $$hunter2$$, $q$secret$q$ WHERE id = $1", false, "SELECT ?, ? WHERE id = $1"},
		{"PostgreSQLUnterminatedDollarQuote", "SELECT $$hunter2", false, "SELECT ?"},
		{"MySQLDollarIsIdentifier", "SELECT c$d, `a$b` FROM t", true, "SELECT c$d, `a$b` FROM t"},
		{"Binary", "flags = 0b1010", true, "flags = ?"},
		// An apostrophe in a comment must not pair with a real quote.
		{"LineComment", "note = 'x' -- it's\n AND ssn = 'SECRET'", false, "note = ? --?\n AND ssn = ?"},
		{"BlockComment", "/* it's */ name = 'John'", true, "/*?*/ name = ?"},
		{"UnterminatedBlockComment", "x = 1 /* it's", true, "x = ? /*?"},
		// A backtick means nothing to PostgreSQL; a stray one must not hide
		// the literals after it.
		{"PostgreSQLBacktick", "SELECT `a FROM t WHERE ssn = 'SECRET'", false, "SELECT `a FROM t WHERE ssn = ?"},
		// PostgreSQL comments nest; MySQL ones end at the first */.
		{"PostgreSQLNestedComment", "SELECT * FROM t /* a /* nested */ secret */ WHERE a = $1", false, "SELECT * FROM t /*?*/ WHERE a = $1"},
		{"MySQLCommentsDoNotNest", "/* a /* b */ x = 'y'", true, "/*?*/ x = ?"},
		{"Empty", "", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, maskSQLLiterals(tc.in, tc.doubleQuoted))
		})
	}
}

func TestMaskMySQLPlan(t *testing.T) {
	t.Parallel()

	plan := `{
  "query_block": {
    "select_id": 1,
    "cost_info": {"query_cost": "2014.25"},
    "table": {
      "table_name": "orders",
      "access_type": "ALL",
      "rows_examined_per_scan": 199937,
      "attached_condition": "((` + "`school`.`orders`.`region`" + ` = 'emea') and (` + "`school`.`orders`.`amount`" + ` < 450.00))"
    }
  }
}`
	got, ok := maskMySQLPlan(plan)
	assert.True(t, ok)
	assert.NotContains(t, got, "emea")
	assert.NotContains(t, got, "450.00")
	assert.Contains(t, got, "`school`.`orders`.`region` = ?")
	assert.Contains(t, got, "`school`.`orders`.`amount` < ?", "comparison operators stay readable, not \\u003c")
	assert.Contains(t, got, `"rows_examined_per_scan": 199937`)
	assert.Contains(t, got, `"query_cost": "2014.25"`, "costs are not under a condition key")
	assert.Less(t, strings.Index(got, "select_id"), strings.Index(got, "cost_info"), "key order is preserved")

	// A backslash never escapes inside an identifier, whatever the reading of
	// string literals; an identifier ending in one must not shift the quotes.
	got, ok = maskMySQLPlan(`{"attached_condition": "((` + "`t`.`dir\\\\`" + ` = 'C:\\\\') and (` + "`t`.`n`" + ` = 'O\\'Brien') and (` + "`t`.`ssn`" + ` = 'SECRET'))"}`)
	assert.True(t, ok)
	assert.NotContains(t, got, "Brien")
	assert.NotContains(t, got, "SECRET")

	// Format version 2 writes a Limit node's LIMIT and OFFSET as numbers.
	got, ok = maskMySQLPlan(`{"access_type": "limit", "limit": 10, "limit_offset": 5, "operation": "Limit/Offset: 10/5 row(s)", ` +
		`"estimated_rows": 3, "inputs": [{"operation": "Sort: orders.amount DESC, limit input to 15 row(s) per chunk", "per_chunk_limit": 15}]}`)
	assert.True(t, ok)
	assert.Contains(t, got, `"limit": "?"`)
	assert.Contains(t, got, `"limit_offset": "?"`)
	assert.Contains(t, got, `"per_chunk_limit": "?"`, "the sum of LIMIT and OFFSET")
	assert.NotContains(t, got, "15")
	assert.Contains(t, got, `"estimated_rows": 3`, "estimates are kept")

	// Names are kept, even those a digit starts, in both format versions.
	got, ok = maskMySQLPlan(`{"query_block": {"table": {"table_name": "2fa_codes", "key": "1st_idx", "possible_keys": ["1st_idx"], ` +
		`"used_key_parts": ["2nd_col"], "used_columns": ["2nd_col", "id"], "attached_condition": "(` + "`t`.`2nd_col`" + ` = 42)"}}, ` +
		`"real_table_name": "2fa_codes", "inputs": [{"table_name": "2fa_codes", "alias": "2f", "schema_name": "3rd_db", "index_name": "1st_idx"}]}`)
	assert.True(t, ok)
	for _, name := range []string{
		`"table_name": "2fa_codes"`, `"key": "1st_idx"`, `"1st_idx"`, `"2nd_col"`,
		`"alias": "2f"`, `"schema_name": "3rd_db"`, `"index_name": "1st_idx"`,
	} {
		assert.Contains(t, got, name)
	}
	assert.Contains(t, got, "`t`.`2nd_col` = ?", "conditions are still masked")
	assert.Contains(t, got, `"real_table_name": "?fa_codes"`, "pmm-agent parses it from the statement text")

	_, ok = maskMySQLPlan("id | select_type | table")
	assert.False(t, ok, "a non-JSON plan is reported, not passed off as masked")

	// explain_json_format_version=2 (MySQL 8.3+) carries literals outside any
	// condition key: the rewritten statement, the operation text, the ranges.
	v2 := `{"query": "/* select#1 */ select ` + "`t`.`id`" + ` from ` + "`t`" + ` where (` + "`t`.`email`" + ` = 'x@y.com')",
  "query_plan": {"operation": "Filter: (` + "`t`.`email`" + ` = 'x@y.com')", "access_type": "filter", "estimated_rows": 10.5,
    "inputs": [{"operation": "Index range scan on t using email", "ranges": ["('a@b' <= email <= 'a@b')"]}]}}`
	got, ok = maskMySQLPlan(v2)
	assert.True(t, ok)
	assert.NotContains(t, got, "x@y.com")
	assert.NotContains(t, got, "a@b")
	assert.Contains(t, got, `"operation": "Filter: (`+"`t`.`email`"+` = ?)"`)
	assert.Contains(t, got, `"operation": "Index range scan on t using email"`)
	assert.Contains(t, got, `"estimated_rows": 10.5`)
}

// TestMaskMySQLPlanWarnings covers the SHOW WARNINGS pmm-agent appends to a JSON
// plan, whose Note 1003 is the rewritten statement with every literal in it.
func TestMaskMySQLPlanWarnings(t *testing.T) {
	t.Parallel()

	plan := `{
  "query_block": {"select_id": 1, "table": {"table_name": "orders", "access_type": "ALL",
    "attached_condition": "(` + "`school`.`orders`.`region`" + ` = 'emea')"}},
  "real_table_name": "orders",
  "warnings": [
    {"Code": 1003, "Level": "Note",
     "Message": "/* select#1 */ select count(0) from ` + "`school`.`orders`" + ` where ((` + "`school`.`orders`.`status`" + ` = 'paid') and (` + "`school`.`orders`.`region`" + ` = 'emea'))"}
  ]
}`
	got, ok := maskMySQLPlan(plan)
	assert.True(t, ok)
	assert.NotContains(t, got, "emea")
	assert.NotContains(t, got, "paid")
	assert.Contains(t, got, `"Code": 1003`, "warning codes are not literals")
	assert.Contains(t, got, `"Level": "Note"`)
	assert.Contains(t, got, "`school`.`orders`.`status` = ?", "the rewritten statement keeps its shape")
	assert.Contains(t, got, `"real_table_name": "orders"`)
}

func TestMaskPGPlan(t *testing.T) {
	t.Parallel()

	t.Run("Text", func(t *testing.T) {
		t.Parallel()

		plan := "Nested Loop  (cost=0.29..16.34 rows=1 width=72)\n" +
			"  Join Filter: (o.customer_id = c.id)\n" +
			"  ->  Seq Scan on customers c  (cost=0.00..458.00 rows=1 width=64)\n" +
			"        Filter: ((email)::text = 'user42@example.com'::text)\n" +
			"  ->  Index Scan using orders_pkey on orders o  (cost=0.29..8.30 rows=1 width=8)\n" +
			"        Index Cond: (id = 42)\n" +
			"  Sort Key: (CASE WHEN ((c.status)::text = 'vip'::text) THEN 1 ELSE 0 END)\n" +
			"  Group Key: date_trunc('month'::text, o.created_at)\n" +
			"  Workers Planned: 2\n" +
			// A newline inside a constant is printed as is.
			"  ->  Seq Scan on notes n  (cost=0.00..1.00 rows=1 width=4)\n" +
			"        Filter: (note = 'line1\nssn 123-45-6789 ''secret'''::text)\n" +
			"  ->  Seq Scan on tags  (cost=0.00..2.00 rows=1 width=4)"
		got := maskPGPlan(plan)
		assert.NotContains(t, got, "user42@example.com")
		assert.Contains(t, got, "Filter: ((email)::text = ?::text)")
		assert.Contains(t, got, "Index Cond: (id = ?)")
		assert.Contains(t, got, "Join Filter: (o.customer_id = c.id)", "identifiers survive")
		assert.Contains(t, got, "(cost=0.00..458.00 rows=1 width=64)", "estimates are not on predicate lines")
		assert.NotContains(t, got, "vip", "every property line is masked, not only predicates")
		assert.NotContains(t, got, "month")
		assert.Contains(t, got, "Workers Planned: 2", "a lone number is a planner count, not a literal")
		assert.NotContains(t, got, "123-45-6789", "a literal's continuation lines are masked too")
		assert.NotContains(t, got, "secret")
		assert.Contains(t, got, "Filter: (note = ?::text)\n  ->  Seq Scan on tags  (cost=0.00..2.00 rows=1 width=4)",
			"the value ends where its quotes balance")

		// A newline inside a quoted identifier is printed as is too.
		got = maskPGPlan("Seq Scan on t\n  Filter: (\"my\ncol\" = 'secret'::text)")
		assert.NotContains(t, got, "secret")

		// The apostrophe of a double-quoted identifier opens no constant.
		got = maskPGPlan("Seq Scan on t\n  Filter: (\"o'neil\" = 'x')\n  ->  Index Scan on i  (cost=1.00..2.00 rows=3 width=4)")
		assert.Equal(t, "Seq Scan on t\n  Filter: (\"o'neil\" = ?)\n  ->  Index Scan on i  (cost=1.00..2.00 rows=3 width=4)", got)
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		plan := `[{"Plan":{"Node Type":"Seq Scan","Total Cost":458.0,"Filter":"((email)::text = 'x@y'::text)",` +
			`"Sort Key":["(CASE WHEN (status = 'vip'::text) THEN 1 ELSE 0 END)"]}}]`
		got := maskPGPlan(plan)
		assert.NotContains(t, got, "x@y")
		assert.NotContains(t, got, "vip")
		assert.Contains(t, got, `"Node Type":"Seq Scan"`)
		assert.Contains(t, got, `"Filter":"((email)::text = ?::text)"`)
		assert.Contains(t, got, `"Total Cost":458.0`)

		got = maskPGPlan(`[{"Plan":{"Node Type":"Index Scan","Relation Name":"2fa_codes","Schema":"3rd","Alias":"2f",` +
			`"Index Name":"1st_idx","Index Cond":"(id = 42)"}}]`)
		for _, name := range []string{`"Relation Name":"2fa_codes"`, `"Schema":"3rd"`, `"Alias":"2f"`, `"Index Name":"1st_idx"`} {
			assert.Contains(t, got, name, "names are kept, even those a digit starts")
		}
		assert.Contains(t, got, `"Index Cond":"(id = ?)"`)
	})

	t.Run("UnparseableJSONIsWithheld", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, planWithheld, maskPGPlan(`{"Plan": "truncated`))
	})
}

func TestMaskMongoPlan(t *testing.T) {
	t.Parallel()

	plan := `{"queryPlanner":{"namespace":"shop.customers","parsedQuery":{"email":{"$eq":"user42@example.com"}},` +
		`"winningPlan":{"stage":"COLLSCAN","filter":{"age":{"$gt":{"$numberInt":"30"}}}}},` +
		`"command":{"find":"customers","filter":{"email":"user42@example.com"},"$db":"shop"}}`
	got := maskMongoPlan(plan)
	assert.NotContains(t, got, "user42@example.com")
	assert.NotContains(t, got, `"30"`)
	assert.Contains(t, got, `"stage":"COLLSCAN"`)
	assert.Contains(t, got, `"namespace":"shop.customers"`)
	assert.Contains(t, got, `"command":{"find":"?",`, "the echoed command is masked whole; the collection shows as namespace")
	assert.Contains(t, got, `"email":{"$eq":"?"}`, "field names and operators survive; only values are masked")

	assert.NotContains(t, got, `"$db":"shop"`, "strings outside plan structure are masked")

	// The slot-based engine prints its constants into "slots" and "stages"
	// strings, and an aggregation's stages can hold literals of their own.
	sbe := `{"queryPlanner":{"winningPlan":{"queryPlan":{"stage":"COLLSCAN","filter":{"email":{"$eq":"u@x"}}},` +
		`"slotBasedPlan":{"slots":"$$RESULT=s5 env: { s2 = \"u@x\" }","stages":"[1] filter {(s1 == \"u@x\")}"}}},` +
		`"stages":[{"$cursor":{"queryPlanner":{"winningPlan":{"stage":"IXSCAN","indexName":"email_1",` +
		`"keyPattern":{"email":{"$numberInt":"1"}}}}}},{"$group":{"_id":"$region","total":{"$sum":{"$numberInt":"977"}}}}]}`
	got = maskMongoPlan(sbe)
	assert.NotContains(t, got, "u@x")
	assert.NotContains(t, got, "977")
	assert.Contains(t, got, `"stage":"IXSCAN"`, "$cursor wraps the plan, not query values")
	assert.Contains(t, got, `"indexName":"email_1"`)
	assert.Contains(t, got, `"keyPattern":{"email":{"$numberInt":"1"}}`, "extended-JSON numbers outside query values are kept")

	// Numbers are masked too, unless they are statistics or an index's key
	// pattern: a find's min and max, and mapReduce's scope, hold query values.
	stats := `{"executionStats":{"nReturned":{"$numberInt":"5"},"totalDocsExamined":200000,` +
		`"executionStages":{"stage":"COLLSCAN","works":200002,"isEOF":true}},` +
		`"command":{"find":"c","min":{"age":{"$numberInt":"30"}},"max":{"age":42}},"scope":{"threshold":12345}}`
	got = maskMongoPlan(stats)
	for _, literal := range []string{"30", "42", "12345"} {
		assert.NotContains(t, got, literal)
	}
	assert.Contains(t, got, `"nReturned":{"$numberInt":"5"}`)
	assert.Contains(t, got, `"totalDocsExamined":200000`)
	assert.Contains(t, got, `"works":200002,"isEOF":true`)

	// A user field named like a plan key keeps no value: plan keys count only
	// outside the query values.
	echoed := `{"command":{"find":"c","min":{"version":{"$numberInt":"3"}},"max":{"count":"secret"},` +
		`"comment":{"host":"secret-comment"},"projection":{"stage":"lit"}},` +
		`"queryPlanner":{"winningPlan":{"stage":"PROJECTION_DEFAULT","transformBy":{"stage":"lit2"}}}}`
	got = maskMongoPlan(echoed)
	for _, literal := range []string{`"3"`, "secret", "lit"} {
		assert.NotContains(t, got, literal)
	}
	assert.Contains(t, got, `"stage":"PROJECTION_DEFAULT"`)

	assert.Equal(t, planWithheld, maskMongoPlan("not json"))
}

func TestRedactPlan(t *testing.T) {
	t.Parallel()

	table := "id | select_type | table | type\n1  | SIMPLE      | orders | ALL"
	assert.Equal(t, planWithheld, redactPlan(engineMySQL, formatJSON, `{"explain_result":"!!"}`, false),
		"an envelope that was not unwrapped still carries the explained statement")
	assert.Equal(t, table, redactPlan(engineMySQL, formatTraditional, table, true), "a traditional table prints no predicates")
	assert.Equal(t, planWithheld, redactPlan(engineMySQL, formatJSON, table, true), "a JSON request that is not JSON is withheld")
	assert.NotContains(t, redactPlan(engineMySQL, formatTraditional, `{"attached_condition":"(a = 'x')"}`, true), "'x'",
		"JSON is masked whatever format was asked for")
	assert.Equal(t, planWithheld, redactPlan("valkey", formatJSON, "anything", true))

	// MySQL NDB Cluster prints a pushed condition in the Extra column.
	ndb := "id |select_type |table |type |rows |Extra\n" +
		"1  |SIMPLE      |t     |ALL  |5000 |Using where with pushed condition: ((`t`.`ssn` = '123-45-6789') or (`t`.`tag` = 'a|b'))"
	got := redactPlan(engineMySQL, formatTraditional, ndb, true)
	assert.NotContains(t, got, "123-45-6789")
	assert.NotContains(t, got, "a|b", "a | inside the condition stays in the masked cell")
	assert.Contains(t, got, "1  |SIMPLE      |t     |ALL  |5000 |Using where with pushed condition: ((`t`.`ssn` = ?) or (`t`.`tag` = ?))")
}

func TestMapActionErrorRedaction(t *testing.T) {
	t.Parallel()

	syntax := "Error 1064 (42000): You have an error in your SQL syntax; near 'emea' AND status = 'paid'' at line 1"
	e := mapActionError(syntax, true)
	assert.Equal(t, codeInvalidInput, e.code)
	assert.NotContains(t, e.Error(), "emea")
	assert.Contains(t, e.Error(), "Error 1064 (42000):", "the error code survives masking")

	mongo := `query: {"ns":"shop.customers","op":"query","query":{"email":"user42@example.com"}}: cannot explain`
	e = mapActionError(mongo, true)
	assert.NotContains(t, e.Error(), "user42@example.com")

	denied := "Error 1142 (42000): SELECT command denied to user 'pmm'@'localhost' for table 'orders'"
	e = mapActionError(denied, true)
	assert.Equal(t, codeInsufficientPrivileges, e.code)
	assert.Contains(t, e.Error(), "'pmm'@'localhost'", "privilege errors keep the user remediation needs")

	missing := "Error 1146 (42S02): Table 'shop.orders' doesn't exist"
	e = mapActionError(missing, true)
	assert.Equal(t, codeNotFound, e.code)
	assert.Contains(t, e.Error(), "Table 'shop.orders' doesn't exist", "an error that names only identifiers is kept")

	// A word inside a quoted statement must not pass for the error's own:
	// classified as a privilege error, the message would go out unmasked.
	perms := `query: {"ns":"app.user_permissions","query":{"email":"user42@example.com"}}: cannot explain`
	e = mapActionError(perms, true)
	assert.NotEqual(t, codeInsufficientPrivileges, e.code)
	assert.NotContains(t, e.Error(), "user42@example.com")

	nearDenied := "Error 1064 (42000): You have an error in your SQL syntax; near 'denied' AND ssn = '123-45-6789'' at line 1"
	e = mapActionError(nearDenied, true)
	assert.Equal(t, codeInvalidInput, e.code)
	assert.NotContains(t, e.Error(), "123-45-6789")

	// "near '...'" quotes the statement with its own quotes unescaped, so the
	// quote pairing flips wherever the fragment does not start on a literal.
	for _, near := range []string{
		"Error 1064 (42000): You have an error in your SQL syntax; near 'WHERE region = 'emea' AND status = 'paid'' at line 1",
		"Error 1064 (42000): You have an error in your SQL syntax; near ''user42@exa' at line 1",
	} {
		e = mapActionError(near, true)
		assert.Equal(t, codeInvalidInput, e.code, near)
		for _, literal := range []string{"emea", "paid", "user42"} {
			assert.NotContains(t, e.Error(), literal, near)
		}
	}

	// An apostrophe must neither swallow the classifying words nor pair with
	// a real quote.
	key := "Error 1176 (42000): Key 'idx' doesn't exist in table 'orders'"
	e = mapActionError(key, true)
	assert.Equal(t, codeNotFound, e.code)
	assert.Contains(t, e.Error(), key)
	// An error of a shape not known to be safe is withheld whole.
	e = mapActionError("can't parse value 'hunter2' here", true)
	assert.NotContains(t, e.Error(), "hunter2")
	assert.Contains(t, e.Error(), withheldMessage)

	// ANSI_QUOTES turns a double-quoted string into a column name, so 1054
	// can quote a value: classified, but withheld.
	e = mapActionError("Error 1054 (42S22): Unknown column 'x@y.com' in 'where clause'", true)
	assert.Equal(t, codeNotFound, e.code)
	assert.NotContains(t, e.Error(), "x@y.com")

	// MongoDB echoes the failed command; the classifying words are outside it.
	unauthorized := "cannot explain this type of query\n" +
		`(Unauthorized) not authorized on shop to execute command { explain: { find: "customers", filter: { email: "x@y" } } }`
	e = mapActionError(unauthorized, true)
	assert.Equal(t, codeInsufficientPrivileges, e.code)
	assert.NotContains(t, e.Error(), "x@y")
	e = mapActionError(`server selection error: context deadline exceeded, current topology: { Type: Unknown, Servers: [{ Addr: db:27017 }] }`, true)
	assert.Equal(t, codeAgentUnreachable, e.code)

	// MongoDB's planner prints the query's values bare.
	planner := "cannot explain this type of query\n(BadValue) error processing query: ns=shop.c Tree: age $eq 42\nplanner returned error"
	e = mapActionError(planner, true)
	assert.NotContains(t, e.Error(), "42")
	assert.Contains(t, e.Error(), "cannot explain this type of query\n(BadValue) (message withheld")

	root := "Error 1698 (28000): Access denied for user 'root'@'localhost'"
	assert.Equal(t, root, mapActionError(root, true).message, "1698 names only a user and a host")

	// A MySQL number outside the table is classified on its text; the
	// message is withheld all the same.
	e = mapActionError("Error 1630 (42000): FUNCTION shop.f does not exist. Check the 'Function Name Parsing' section", true)
	assert.Equal(t, codeNotFound, e.code)
	assert.Contains(t, e.Error(), "Error 1630 (42000): (message withheld")

	// MongoDB code names can hold digits.
	location := "cannot explain this type of query\n(Location17276) Use of undefined variable: secretvar and value 424242"
	e = mapActionError(location, true)
	assert.NotContains(t, e.Error(), "secretvar")
	assert.NotContains(t, e.Error(), "424242")
	assert.Contains(t, e.Error(), "cannot explain this type of query\n(Location17276) "+withheldMessage)

	// A login failure arrives inside a connection error: it names hosts, not
	// statements, so it is kept, and it is a credentials problem.
	login := "cannot explain this type of query\nconnection(db:27017[-1]) error occurred during connection handshake: auth error: " +
		`sasl conversation error: unable to authenticate using mechanism "SCRAM-SHA-256": (AuthenticationFailed) Authentication failed.`
	e = mapActionError(login, true)
	assert.Equal(t, codeInsufficientPrivileges, e.code)
	assert.Equal(t, login, e.message)

	// A value cannot choose the class of an unknown number.
	e = mapActionError("Error 1292 (22007): Truncated incorrect DOUBLE value: 'connection'", true)
	assert.Equal(t, codePMMUnavailable, e.code)
	assert.NotContains(t, e.Error(), "'connection'")

	// pmm-agent's own texts and network errors quote no statement.
	for _, kept := range []string{
		"query to EXPLAIN is empty",
		"dial tcp 10.0.0.5:3306: connect: connection refused",
		"database name is not included in this query. Explain could not be triggered without this info: Error 1046 (3D000): No database selected",
	} {
		assert.Equal(t, kept, mapActionError(kept, true).message)
	}

	// A query example that does not parse leaves its brace open.
	broken := `query: {"ns":"shop.customers","query":{"email":"SECRET-9911@example.com": error decoding key query: invalid JSON input`
	assert.NotContains(t, mapActionError(broken, true).Error(), "SECRET-9911")

	assert.Contains(t, mapActionError(syntax, false).Error(), "emea", "raw SQL on leaves the message verbatim")
}
