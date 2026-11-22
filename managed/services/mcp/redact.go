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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// This file masks literal values in execution plans and agent errors when
// PMM_MCP_RAW_SQL is off. Plans carry customer data wherever they echo a
// predicate or a statement, and where that is differs by engine and version,
// so masking is fail-closed: every string is masked unless it is known to hold
// plan structure only. Literals become ?, the way QAN fingerprints do, so the
// plan keeps its shape - access type, key, rows - and stays useful for triage.
// Anything that cannot be parsed is withheld rather than passed through.

// planWithheld replaces a plan body that could not be parsed for masking.
const planWithheld = "(plan withheld: it could not be parsed to mask literal values, and PMM_MCP_RAW_SQL is off)"

// statementWithheld replaces the rewritten statement MySQL puts in a plan.
const statementWithheld = "(statement withheld: MySQL's rewritten statement can quote values, and PMM_MCP_RAW_SQL is off)"

// mongoLiteralKeys are the MongoDB explain keys whose subtrees hold query
// values, numbers included: the echoed command whole, with its filter,
// bounds, comment and projection, and the parsed filter, index bounds,
// pipelines, update specifications and projections of the plan. Keys starting
// with $ (operators and aggregation stages) are treated the same way; see
// mongoLiteralKey. Field names are keys and survive; only the values beneath
// are masked, so a field named like a plan key (version, count) cannot keep
// its value.
var mongoLiteralKeys = []string{
	"command", "comment", "projection", "transformBy",
	"parsedQuery", "filter", "indexBounds", "pipeline", "query", "q", "u", "update", "updates", "let",
}

// mongoPlanKeys are the MongoDB explain keys whose values are plan structure
// or statistics, never a query value: stage, index and collection names, plan
// hashes, server version, and the execution counters. They are kept only on
// the plan's own structure (mongoPlanContainers), as is an index's keyPattern.
// Every other string and number is masked, including the slot-based engine's
// "slots" and "stages" text and the bounds of a find's min and max.
var mongoPlanKeys = []string{
	"stage", "indexName", "namespace", "direction", "queryFramework", "explainVersion",
	"planCacheKey", "planCacheShapeHash", "queryHash", "queryShapeHash",
	"host", "port", "version", "gitVersion",
	"nReturned", "executionTimeMillis", "executionTimeMillisEstimate", "totalKeysExamined", "totalDocsExamined",
	"keysExamined", "docsExamined", "works", "advanced", "needTime", "needYield", "saveState", "restoreState",
	"isEOF", "seeks", "dupsTested", "dupsDropped", "indexVersion", "ok",
}

// mongoPlanContainers are the MongoDB explain keys whose objects are the plan's
// own structure. Any other object, such as a compound text index's indexPrefix
// with the query's values under their field names, is masked whole.
var mongoPlanContainers = []string{
	"queryPlanner", "winningPlan", "rejectedPlans", "queryPlan", "slotBasedPlan", "inputStage", "inputStages",
	"executionStats", "executionStages", "allPlansExecution", "serverInfo", "shards", "stages", "$cursor",
}

// pgEstimates matches the trailing (cost=...) and (actual ...) groups of a
// PostgreSQL text-plan node line, whose numbers are estimates and measurements.
var pgEstimates = regexp.MustCompile(`(?:\s+\((?:cost=[0-9.]+\.\.[0-9.]+ rows=[0-9.]+ width=[0-9]+|` +
	`actual (?:time=[0-9.]+\.\.[0-9.]+ )?rows=[0-9.]+ loops=[0-9]+)\))+$`)

// mysqlValueNumbers are the JSON plan keys whose numbers come from the query's
// own values rather than estimates, in format version 2: a Limit node's LIMIT
// and OFFSET, and a sort's per_chunk_limit, which is their sum (live-checked
// on 8.4.11).
var mysqlValueNumbers = []string{"limit", "limit_offset", "per_chunk_limit"}

// mysqlIdentifierKeys are the JSON plan keys that hold only names, which masking
// would mangle.
var mysqlIdentifierKeys = []string{
	"table_name", "alias", "schema_name", "key", "index_name", "possible_keys", "used_key_parts", "used_columns",
}

// pgIdentifierKeys are the JSON plan keys whose strings are only names.
var pgIdentifierKeys = []string{"Relation Name", "Schema", "Alias", "Index Name", "CTE Name"}

// mysqlOwnName matches the names MySQL gives its own tables and keys in a plan, such as <derived2>, <union1,2>,
// <union temporary> and <auto_key0>.
var mysqlOwnName = regexp.MustCompile(`^<(?:(?:derived|subquery|union|intersect|except|auto_key)[0-9]+(?:,[0-9]+)*|` +
	`(?:union |intersect |except )?temporary|auto_distinct_key|group_key|hash_distinct_key)>$`)

// mysqlNumber matches the numeric strings MySQL writes into a JSON plan -
// costs, "filtered", data sizes such as "1K" - which are kept.
var mysqlNumber = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?[KMGTPE]?$`)

// mysqlErrorHeader matches the "Error 1146 (42S02): " prefix the MySQL driver
// puts on a server error. It never carries a literal and is kept.
var mysqlErrorHeader = regexp.MustCompile(`^Error ([0-9]+)(?: \([0-9A-Z]{5}\))?: `)

// mongoErrorName matches the "(BadValue) " or "(Location17276) " the MongoDB
// driver puts before a server error's message.
var mongoErrorName = regexp.MustCompile(`^\(([A-Za-z0-9]+)\) `)

// mongoExplainPrefix is the text pmm-agent's MongoDB explain action joins
// before a server error (errCannotExplain in
// agent/runner/actions/mongodb_explain_action.go).
const mongoExplainPrefix = "cannot explain this type of query\n"

// safeActionError matches the action errors that cannot quote the statement:
// pmm-agent's own fixed texts (agent/runner/actions), and network errors,
// which name only hosts and ports. Any other error message is withheld when
// raw SQL is off.
var safeActionError = regexp.MustCompile(`^(?:` +
	`query to EXPLAIN is empty|` +
	`query EXPLAIN failed because the query exceeded max length and got trimmed\. Set max-query-length to a larger value|` +
	`query EXPLAIN functionality is supported only for DML queries - SELECT, INSERT, UPDATE, DELETE and REPLACE|` +
	`cannot JSON encode the explain response|` +
	`unsupported output format [A-Za-z_]+|` +
	`(?:command|operation) [A-Za-z]+ is not supported for explain|` +
	`database name is not included in this query\. Explain could not be triggered without this info: Error 1046 \(3D000\): No database selected` +
	`)$|^(?:dial tcp |server selection error: |connection\([^)]*\) error occurred during connection handshake: )`)

// maskSQLLiterals replaces string and numeric literals in a SQL fragment with ?.
//
// Identifiers are copied through untouched: backquoted ones when
// doubleQuotedStrings is set, as in MySQL, and double-quoted ones when it is
// not, as in PostgreSQL. A digit that continues an identifier (col1, $1) is
// not a literal. The text of a -- or /* */ comment, and in MySQL of a #
// comment, is masked: it can quote anything, and an apostrophe in it must not
// pair with a real quote.
//
// Whether a backslash escapes a quote in a string depends on server settings
// (MySQL's NO_BACKSLASH_ESCAPES, PostgreSQL's standard_conforming_strings), so
// the text is scanned both ways and a byte is masked when either reading puts
// it in a literal: guessing wrong then over-masks instead of leaking the next
// literal.
func maskSQLLiterals(s string, doubleQuotedStrings bool) string {
	masked, _ := maskSQL(s, doubleQuotedStrings)
	return masked
}

// maskSQL is maskSQLLiterals that also reports whether s holds a quoted or
// dollar-quoted string outside its comments, under either backslash reading.
func maskSQL(s string, doubleQuotedStrings bool) (string, bool) {
	marks := sqlMarks(s, doubleQuotedStrings, false)
	return renderMarks(s, marks), slices.Contains(marks, stringMark)
}

// maskMySQLPlanText masks a text of a MySQL plan like maskSQLLiterals, and the
// text of every backquoted name that is not a plain name: MySQL names an
// unaliased select-list expression after its own text, literals included.
func maskMySQLPlanText(s string) string {
	return renderMarks(s, sqlMarks(s, true, true))
}

// renderMarks writes s with each run of marked bytes replaced by one ?.
func renderMarks(s string, marks []byte) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := range len(s) {
		switch {
		case marks[i] == plainMark:
			b.WriteByte(s[i])
		case i == 0 || marks[i-1] == plainMark:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// Marks for the bytes of a SQL fragment; a higher one outranks a lower.
const (
	plainMark byte = iota
	nameMark
	commentMark
	numberMark
	stringMark
)

// sqlMarks marks the bytes of s that either backslash reading puts in a
// literal or a comment, and with plainNames the text of each quoted
// identifier that is not a plain name.
func sqlMarks(s string, doubleQuotedStrings, plainNames bool) []byte {
	marks := scanSQL(s, doubleQuotedStrings, true, plainNames)
	for i, m := range scanSQL(s, doubleQuotedStrings, false, plainNames) {
		marks[i] = max(marks[i], m)
	}
	return marks
}

// scanSQL marks the bytes of s that belong to a literal or a comment under one
// backslash reading, and with plainNames the text of each quoted identifier
// that is not a plain name. Identifiers are never backslash-escaped, whatever
// the reading; one that does not close is marked as a string to the end of s.
func scanSQL(s string, doubleQuotedStrings, backslashEscapes, plainNames bool) []byte {
	marks := make([]byte, len(s))
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '-' && strings.HasPrefix(s[i:], "--"):
			i = markRange(marks, i+len("--"), lineEnd(s, i), commentMark)
		case c == '#' && doubleQuotedStrings:
			i = markRange(marks, i+len("#"), lineEnd(s, i), commentMark)
		case c == '/' && strings.HasPrefix(s[i:], "/*"):
			from := i + len("/*")
			end := blockCommentEnd(s, from, !doubleQuotedStrings)
			if end < 0 {
				i = markRange(marks, from, len(s), commentMark)
				continue
			}
			i = markRange(marks, from, end, commentMark) + len("*/")
		case (c == '`' && doubleQuotedStrings) || (c == '"' && !doubleQuotedStrings):
			end, closed := quotedClose(s, i, false)
			switch {
			case !closed:
				i = markRange(marks, i, len(s), stringMark)
			case plainNames && !isPlainName(s[i+1:end-1]):
				i = markRange(marks, i+1, end-1, nameMark) + 1
			default:
				i = end
			}
		case c == '\'' || c == '"':
			i = markRange(marks, i, quotedEnd(s, i, backslashEscapes), stringMark)
		case isDigit(c) && (i == 0 || !isIdentByte(s[i-1])):
			i = markRange(marks, i, numberEnd(s, i), numberMark)
		case c == '$' && !doubleQuotedStrings && (i == 0 || !isIdentByte(s[i-1])):
			j, ok := dollarQuoteEnd(s, i)
			if !ok {
				i++
				continue
			}
			i = markRange(marks, i, j, stringMark)
		default:
			i++
		}
	}
	return marks
}

// blockCommentEnd returns the index of the */ that closes the block comment
// whose text starts at s[from], or -1. PostgreSQL comments nest, MySQL ones do
// not.
func blockCommentEnd(s string, from int, nested bool) int {
	depth := 1
	for i := from; i+1 < len(s); i++ {
		switch {
		case nested && s[i] == '/' && s[i+1] == '*':
			depth++
			i++
		case s[i] == '*' && s[i+1] == '/':
			depth--
			if depth == 0 {
				return i
			}
			i++
		}
	}
	return -1
}

// markRange marks marks[from:to] and returns to.
func markRange(marks []byte, from, to int, mark byte) int {
	for i := from; i < to; i++ {
		marks[i] = mark
	}
	return to
}

// lineEnd returns the index of the newline that ends the line holding s[i],
// or len(s).
func lineEnd(s string, i int) int {
	j := strings.IndexByte(s[i:], '\n')
	if j < 0 {
		return len(s)
	}
	return i + j
}

// dollarQuoteEnd returns the index just past the PostgreSQL dollar-quoted
// string ($$...$$ or $tag$...$tag$) that starts at s[start], or false when
// s[start] opens none, as in the $1 placeholder. An unterminated string
// extends to the end of s. A tag follows identifier rules, so its letters
// include every non-ASCII byte.
func dollarQuoteEnd(s string, start int) (int, bool) {
	k := start + 1
	for k < len(s) && s[k] != '$' && (isIdentByte(s[k]) || s[k] >= 0x80) && (k > start+1 || !isDigit(s[k])) {
		k++
	}
	if k >= len(s) || s[k] != '$' {
		return 0, false
	}
	delim := s[start : k+1]
	end := strings.Index(s[k+1:], delim)
	if end < 0 {
		return len(s), true
	}
	return k + 1 + end + len(delim), true
}

// quotedEnd returns the index just past the quoted run that starts at s[start].
// A doubled quote does not end it, nor, when backslashEscapes is set, a
// backslash-escaped one; an unterminated run extends to the end of s.
func quotedEnd(s string, start int, backslashEscapes bool) int {
	end, _ := quotedClose(s, start, backslashEscapes)
	return end
}

// quotedClose is quotedEnd that also reports whether the run closes.
func quotedClose(s string, start int, backslashEscapes bool) (int, bool) {
	q := s[start]
	for j := start + 1; j < len(s); j++ {
		switch {
		case s[j] == '\\' && backslashEscapes:
			j++
		case s[j] == q:
			if j+1 < len(s) && s[j+1] == q {
				j++
				continue
			}
			return j + 1, true
		}
	}
	return len(s), false
}

// numberEnd returns the index just past the numeric literal that starts at
// s[start]: decimal, fractional or exponent, or after 0x, 0o or 0b
// hexadecimal, octal or binary. Underscores between digits, which
// PostgreSQL 16 takes, are part of it.
func numberEnd(s string, start int) int {
	j := start
	if s[j] == '0' && j+1 < len(s) && strings.IndexByte("xXoObB", s[j+1]) >= 0 {
		j += 2
		for j < len(s) && (isHexDigit(s[j]) || s[j] == '_') {
			j++
		}
		return j
	}
	j = digitsEnd(s, j, true)
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		k := j + 1
		if k < len(s) && (s[k] == '+' || s[k] == '-') {
			k++
		}
		if k < len(s) && isDigit(s[k]) {
			j = digitsEnd(s, k, false)
		}
	}
	return j
}

// digitsEnd returns the index just past the digits from s[j], with the
// underscores between them and, with point set, decimal points.
func digitsEnd(s string, j int, point bool) int {
	for j < len(s) && (isDigit(s[j]) || point && s[j] == '.' || s[j] == '_' && j+1 < len(s) && isDigit(s[j+1])) {
		j++
	}
	return j
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isIdentByte(c byte) bool {
	return c == '_' || c == '$' || isDigit(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isPlainName reports whether a name is a plain name, of letters, digits, _ and $, that is not a number such as 42 or
// 0x1F, which MySQL names a column of that constant after.
func isPlainName(name string) bool {
	if name == "" || isDigit(name[0]) && numberEnd(name, 0) == len(name) {
		return false
	}
	for _, r := range name {
		if r != '_' && r != '$' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// maskMySQLPlan masks literals in a MySQL EXPLAIN FORMAT=JSON plan. Every
// string value is masked: literals sit in the *condition keys and in version
// 2's "operation" and "ranges", and in a name MySQL gave a select-list
// expression after its text. The rewritten statement, format version 2's
// "query" and Note 1003 of the SHOW WARNINGS that pmm-agent appends, is
// withheld, and pmm-agent's real_table_name, a word of the statement, is left
// out. Plain names and MySQL's own names in the keys that hold only names, and
// the numeric strings MySQL writes for costs, are kept. It reports false when
// the plan is not JSON.
func maskMySQLPlan(plan string) (string, bool) {
	var out []byte
	members, err := jsonObjectMembers([]byte(plan))
	if err == nil {
		out, err = maskMySQLMembers(members)
	} else {
		out, err = rewriteJSON([]byte(plan), nil, maskMySQLValue)
	}
	if err != nil {
		return "", false
	}
	var indented bytes.Buffer
	err = json.Indent(&indented, out, "", "  ")
	if err != nil {
		return "", false
	}
	return indented.String(), true
}

// maskMySQLMembers masks the top-level members of a MySQL plan, where pmm-agent adds warnings and real_table_name.
func maskMySQLMembers(members []jsonMember) ([]byte, error) {
	w := newJSONWriter()
	w.out.WriteByte('{')
	for _, m := range members {
		if m.key == "real_table_name" {
			continue
		}
		var value []byte
		var err error
		if m.key == "warnings" {
			value, err = maskMySQLWarnings(m.value)
		} else {
			value, err = rewriteJSON(m.value, nil, func(path []string, v any) any {
				return maskMySQLValue(append([]string{m.key}, path...), v)
			})
		}
		if err != nil {
			return nil, err
		}
		if w.out.Len() > 1 {
			w.out.WriteByte(',')
		}
		err = w.scalar(m.key)
		if err != nil {
			return nil, err
		}
		w.out.WriteByte(':')
		w.out.Write(value)
	}
	w.out.WriteByte('}')
	return w.out.Bytes(), nil
}

// maskMySQLValue masks one scalar of a MySQL plan; path holds the keys on the way to it.
func maskMySQLValue(path []string, v any) any {
	key := ""
	if len(path) > 0 {
		key = path[len(path)-1]
	}
	switch v := v.(type) {
	case json.Number:
		if slices.Contains(mysqlValueNumbers, key) {
			return "?"
		}
	case string:
		switch {
		case slices.Contains(path, "query"):
			return statementWithheld
		case slices.Contains(mysqlIdentifierKeys, key):
			if isPlainName(v) || mysqlOwnName.MatchString(v) {
				return v
			}
			return "?"
		case !mysqlNumber.MatchString(v):
			return maskMySQLPlanText(v)
		}
	}
	return v
}

// maskMySQLWarnings masks the SHOW WARNINGS rows pmm-agent appends to a plan, objects with Level, Code and Message.
// The message of Note 1003 is the rewritten statement and is withheld, as is any message whose code is not a number
// or not there; the other messages are masked. A row that is not an object is masked whole.
func maskMySQLWarnings(raw json.RawMessage) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var warnings any
	err := dec.Decode(&warnings)
	if err != nil {
		return nil, err
	}
	rows, ok := warnings.([]any)
	switch {
	case warnings == nil:
	case !ok:
		warnings = "?"
	default:
		for i, row := range rows {
			fields, ok := row.(map[string]any)
			if !ok {
				rows[i] = "?"
				continue
			}
			code, _ := fields["Code"].(json.Number)
			for k, v := range fields {
				switch {
				case k == "Code" && code != "":
				case k == "Message" && (code == "" || code == "1003"):
					fields[k] = statementWithheld
				default:
					fields[k] = maskMySQLValue([]string{"warnings", k}, v)
				}
			}
		}
	}
	w := newJSONWriter()
	err = w.enc.Encode(warnings)
	if err != nil {
		return nil, err
	}
	return bytes.TrimRight(w.scratch.Bytes(), "\n"), nil
}

// maskMongoPlan masks every value beneath the keys that hold query values, and
// every string and number that is not a mongoPlanKeys value or a keyPattern on
// the plan's own structure, keeping stage names, index names, statistics and
// the field names of the filter. Booleans
// and nulls carry no data and are kept. A command's comment, and a $literal or
// $const value, is masked whole.
func maskMongoPlan(plan string) string {
	out, err := rewriteJSON([]byte(plan), mongoOpaque, func(path []string, v any) any {
		if len(path) == 0 || slices.ContainsFunc(path, mongoLiteralKey) {
			return "?"
		}
		switch v.(type) {
		case bool, nil:
			return v
		}
		// An extended-JSON number ({"$numberInt": "1"}) belongs to its parent key.
		keys := path
		if strings.HasPrefix(path[len(path)-1], "$number") && len(path) > 1 {
			keys = path[:len(path)-1]
		}
		key, parents := keys[len(keys)-1], keys[:len(keys)-1]
		if i := slices.Index(path, "keyPattern"); i >= 0 && onMongoPlan(path[:i]) {
			return v
		}
		if slices.Contains(mongoPlanKeys, key) && onMongoPlan(parents) {
			return v
		}
		return "?"
	})
	if err != nil {
		return planWithheld
	}
	return string(out)
}

// onMongoPlan reports whether keys lead through the plan's own structure only.
func onMongoPlan(keys []string) bool {
	return !slices.ContainsFunc(keys, func(k string) bool { return !slices.Contains(mongoPlanContainers, k) })
}

// mongoOpaque reports whether path ends at a value whose keys are data too: a command's comment, which an application
// can fill with anything, or a $literal, which the parsed pipeline shows as $const.
func mongoOpaque(path []string) bool {
	n := len(path)
	if n > 0 && (path[n-1] == "$literal" || path[n-1] == "$const") {
		return true
	}
	return n > 1 && path[n-1] == "comment" && (path[n-2] == "command" || path[n-2] == "originatingCommand")
}

// mongoLiteralKey reports whether a key's subtree holds query values: a
// mongoLiteralKeys entry, or any operator or aggregation stage such as $group
// or $project. Two kinds of $ keys are exempt: $cursor, which wraps the plan
// itself, and the extended-JSON number wrappers such as $numberInt.
func mongoLiteralKey(k string) bool {
	return slices.Contains(mongoLiteralKeys, k) || strings.HasPrefix(k, "$") && k != "$cursor" && !strings.HasPrefix(k, "$number")
}

// maskPGPlan masks literals in a pg_stat_monitor stored plan. A text plan is
// masked as one text, so a constant that spans lines is one literal, and only
// the trailing estimate groups of its node lines keep their numbers. A JSON
// plan is masked in every string value but names: its numbers are estimates,
// and its literals sit in expressions.
func maskPGPlan(plan string) string {
	trimmed := strings.TrimSpace(plan)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		out, err := rewriteJSON([]byte(trimmed), nil, func(path []string, v any) any {
			str, ok := v.(string)
			if !ok || len(path) > 0 && slices.Contains(pgIdentifierKeys, path[len(path)-1]) {
				return v
			}
			return maskSQLLiterals(str, false)
		})
		if err != nil {
			return planWithheld
		}
		return string(out)
	}

	marks := sqlMarks(plan, false, false)
	for start := 0; start < len(plan); {
		end := lineEnd(plan, start)
		loc := pgEstimates.FindStringIndex(plan[start:end])
		if loc != nil {
			for i := start + loc[0]; i < start+loc[1]; i++ {
				if marks[i] == numberMark {
					marks[i] = plainMark
				}
			}
		}
		start = end + 1
	}
	return renderMarks(plan, marks)
}

// maskMongoFingerprint masks the values in a MongoDB fingerprint, a shell call
// with JSON arguments such as db.orders.aggregate([...]): pmm-agent leaves
// the values of aggregation stages other than $match in place. A string
// followed by a colon is a key and is kept, like the "?" of a masked value;
// every other string, and every number, is masked. The value of a $literal or
// $const key, and of a runCommand's comment, is masked whole, keys included.
func maskMongoFingerprint(fingerprint string) string {
	command := strings.HasPrefix(fingerprint, "db.runCommand(")
	var b strings.Builder
	b.Grow(len(fingerprint))
	depth := 0
	for i := 0; i < len(fingerprint); {
		c := fingerprint[i]
		switch {
		case c == '"':
			j := quotedEnd(fingerprint, i, true)
			value := len(fingerprint) - len(strings.TrimLeft(fingerprint[j:], " "))
			key := strings.HasPrefix(fingerprint[value:], ":")
			name := fingerprint[i:j]
			switch {
			case key && (name == `"$literal"` || name == `"$const"` || command && depth == 1 && name == `"comment"`):
				value = len(fingerprint) - len(strings.TrimLeft(fingerprint[value+1:], " "))
				b.WriteString(fingerprint[i:value])
				if value < len(fingerprint) {
					b.WriteByte('?')
					value = jsonValueEnd(fingerprint, value)
				}
				j = value
			case key || name == `"?"`:
				b.WriteString(name)
			default:
				b.WriteByte('?')
			}
			i = j
		case isDigit(c) && (i == 0 || !isIdentByte(fingerprint[i-1])):
			b.WriteByte('?')
			i = numberEnd(fingerprint, i)
		default:
			switch c {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// jsonValueEnd returns the index just past the JSON value that starts at s[start]: a string, a brace or bracket
// group with what it nests, or a bare word or number. A string or group that does not close, or closes with the wrong
// bracket, extends to the end of s.
func jsonValueEnd(s string, start int) int {
	switch s[start] {
	case '"':
		return quotedEnd(s, start, true)
	case '{', '[':
		var closers []byte
		for i := start; i < len(s); i++ {
			switch s[i] {
			case '"':
				i = quotedEnd(s, i, true) - 1
			case '{':
				closers = append(closers, '}')
			case '[':
				closers = append(closers, ']')
			case '}', ']':
				if s[i] != closers[len(closers)-1] {
					return len(s)
				}
				closers = closers[:len(closers)-1]
				if len(closers) == 0 {
					return i + 1
				}
			}
		}
		return len(s)
	default:
		end := start
		for end < len(s) && !strings.ContainsRune(",}]) ", rune(s[end])) {
			end++
		}
		return end
	}
}

// singleQuotedEnd returns the index just past the single-quoted span that
// starts at s[start]. It ends at the next quote not followed by a letter or
// digit, so an apostrophe inside it ('it's here') does not end it; an
// unterminated span extends to the end of s.
func singleQuotedEnd(s string, start int) int {
	for j := start + 1; j < len(s); j++ {
		if s[j] == '\'' && (j+1 == len(s) || !isIdentByte(s[j+1])) {
			return j + 1
		}
	}
	return len(s)
}

// maskTraditionalExtra masks the Extra column of pmm-agent's traditional
// EXPLAIN table, the last of its |-separated cells: MySQL NDB Cluster prints a
// pushed condition there, with its literals. The other columns name tables
// and keys and count rows. The table is split from the left, so a | inside
// the condition stays in the masked cell. A line with fewer cells than the
// header, left by a cell with a line break, withholds the plan.
func maskTraditionalExtra(table string) string {
	lines := strings.Split(table, "\n")
	columns := strings.Count(lines[0], "|") + 1
	// A one-column body, such as the tree format, is masked as one text, so a literal across lines stays one literal.
	if columns == 1 {
		return maskMySQLPlanText(table)
	}
	for i, line := range lines {
		cells := strings.SplitN(line, "|", columns)
		if len(cells) < columns {
			return planWithheld
		}
		cells[len(cells)-1] = maskMySQLPlanText(cells[len(cells)-1])
		lines[i] = strings.Join(cells, "|")
	}
	return strings.Join(lines, "\n")
}

// maskQuoted replaces every quoted string and every brace or bracket group of
// an error message with ?. That is where a MongoDB or pmm-agent error quotes a
// statement, a command or a value: the MongoDB action's "query: <example>",
// MongoDB's echo of a failed command. A single quote that follows a letter or
// digit is an apostrophe (doesn't) and is kept. A group that does not close
// masks the rest of the text.
func maskQuoted(msg string) string {
	var b strings.Builder
	b.Grow(len(msg))
	depth := 0
	for i := 0; i < len(msg); i++ {
		c := msg[i]
		switch {
		case c == '"':
			if depth == 0 {
				b.WriteByte('?')
			}
			i = quotedEnd(msg, i, true) - 1
		case c == '\'' && (i == 0 || !isIdentByte(msg[i-1])):
			if depth == 0 {
				b.WriteByte('?')
			}
			i = singleQuotedEnd(msg, i) - 1
		case c == '{' || c == '[':
			if depth == 0 {
				b.WriteByte('?')
			}
			depth++
		case (c == '}' || c == ']') && depth > 0:
			depth--
		case depth == 0:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// jsonMember is one member of a JSON object, its value not decoded.
type jsonMember struct {
	key   string
	value json.RawMessage
}

// jsonObjectMembers splits a JSON document that is one object into its members, in order.
func jsonObjectMembers(doc []byte) ([]jsonMember, error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, errors.New("not a JSON object")
	}
	var members []jsonMember
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, errors.New("non-string object key")
		}
		var value json.RawMessage
		err = dec.Decode(&value)
		if err != nil {
			return nil, err
		}
		members = append(members, jsonMember{key: key, value: value})
	}
	_, err = dec.Token()
	if err != nil {
		return nil, err
	}
	_, err = dec.Token()
	if !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after JSON value")
	}
	return members, nil
}

// rewriteJSON re-emits a JSON document compactly with every scalar leaf passed
// through leaf, keeping key order. The leaf function receives the object keys on the way to
// the value (array positions are not part of the path) and the value itself:
// a string, json.Number, bool or nil. A value whose path a non-nil opaque reports is written as "?" whole, keys included.
func rewriteJSON(doc []byte, opaque func(path []string) bool, leaf func(path []string, v any) any) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()

	w := newJSONWriter()
	err := rewriteValue(dec, w, nil, opaque, leaf)
	if err != nil {
		return nil, err
	}
	_, err = dec.Token()
	if !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after JSON value")
	}
	return w.out.Bytes(), nil
}

// jsonWriter collects rewriteJSON's output, with one scalar encoder reused
// for every leaf.
type jsonWriter struct {
	out     bytes.Buffer
	scratch bytes.Buffer
	enc     *json.Encoder
}

func newJSONWriter() *jsonWriter {
	w := &jsonWriter{}
	w.enc = json.NewEncoder(&w.scratch)
	w.enc.SetEscapeHTML(false)
	return w
}

func rewriteValue(dec *json.Decoder, w *jsonWriter, path []string, opaque func([]string) bool, leaf func([]string, any) any) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	delim, isDelim := tok.(json.Delim)
	if !isDelim {
		return w.scalar(leaf(path, tok))
	}

	switch delim {
	case '{':
		w.out.WriteByte('{')
		for first := true; dec.More(); first = false {
			keyTok, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyTok.(string)
			if !ok {
				return errors.New("non-string object key")
			}
			if !first {
				w.out.WriteByte(',')
			}
			err = w.scalar(key)
			if err != nil {
				return err
			}
			w.out.WriteByte(':')
			err = rewriteMember(dec, w, append(path, key), opaque, leaf)
			if err != nil {
				return err
			}
		}
		w.out.WriteByte('}')
	case '[':
		w.out.WriteByte('[')
		for first := true; dec.More(); first = false {
			if !first {
				w.out.WriteByte(',')
			}
			err := rewriteValue(dec, w, path, opaque, leaf)
			if err != nil {
				return err
			}
		}
		w.out.WriteByte(']')
	default:
		return errors.New("unexpected JSON delimiter")
	}

	// Consume the closing delimiter.
	_, err = dec.Token()
	return err
}

// rewriteMember writes the value of the object member at path, as "?" whole when opaque reports the path.
func rewriteMember(dec *json.Decoder, w *jsonWriter, path []string, opaque func([]string) bool, leaf func([]string, any) any) error {
	if opaque == nil || !opaque(path) {
		return rewriteValue(dec, w, path, opaque, leaf)
	}
	var skipped json.RawMessage
	err := dec.Decode(&skipped)
	if err != nil {
		return err
	}
	return w.scalar("?")
}

// scalar writes one scalar without HTML escaping, so the < and > of plan
// predicates stay readable.
func (w *jsonWriter) scalar(v any) error {
	if n, ok := v.(json.Number); ok {
		w.out.WriteString(n.String())
		return nil
	}
	w.scratch.Reset()
	err := w.enc.Encode(v)
	if err != nil {
		return err
	}
	w.out.Write(bytes.TrimRight(w.scratch.Bytes(), "\n"))
	return nil
}

// redactPlan masks the literals in a live EXPLAIN result.
//
// The decoded flag reports whether a MySQL result was unwrapped from its envelope. One
// that was not still carries the explained statement, so it is withheld. A
// JSON plan is masked whatever format was asked for; a plan that is not JSON
// and was asked for as traditional is pmm-agent's table, masked in its Extra
// column. Anything else is withheld.
func redactPlan(engine, format, plan string, decoded bool) string {
	switch engine {
	case engineMySQL:
		if !decoded {
			return planWithheld
		}
		masked, ok := maskMySQLPlan(plan)
		if ok {
			return masked
		}
		if format == formatTraditional {
			return maskTraditionalExtra(plan)
		}
		return planWithheld
	case engineMongoDB:
		return maskMongoPlan(plan)
	default:
		return planWithheld
	}
}
