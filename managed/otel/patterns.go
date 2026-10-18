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

package otel

import "strings"

// patternRules replace the variable parts of a line, in order: they unquote logfmt msg values (also one nested,
// escaped level) so the message text stays, then mask quoted strings, timestamps, UUIDs, hex IDs, IPv4 addresses,
// underscore-joined numbers, numbers with a unit and whole numbers, and collapse lists of masked values.
// A number inside a word (md5, rs103) is part of the word. Replacements use ClickHouse's \1 back-references.
var patternRules = [][2]string{
	{`\bmsg="((?:[^"\\]|\\.)*)"`, `msg=\1`},
	{`\bmsg=\\"((?:[^"\\]|\\[^"])*)\\"`, `msg=\1`},
	{`'[^']*'|"[^"]*"`, `'?'`},
	{`\b[0-9]{4}[-/.][0-9]{2}[-/.][0-9]{2}[T ][0-9]{2}:[0-9]{2}(:[0-9]{2}(\.[0-9]+)?)?(Z|[+-][0-9]{2}:?[0-9]{2})?`, `?`},
	{`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, `?`},
	{`\b[0-9a-fA-F]{8,}\b`, `?`},
	{`\b[0-9]{1,3}(\.[0-9]{1,3}){3}\b`, `?`},
	{`\b[0-9]+(_[0-9]+)+\b`, `?`},
	{`\b[0-9]+(\.[0-9]+)?[a-zA-Z]{1,3}\b`, `?`},
	{`\b[0-9]+(\.[0-9]+)?\b`, `?`},
	{`'\?'(\s*,\s*'\?')+`, `'?'`},
	{`\?(\s*,\s*\?)+`, `?`},
}

// PatternExpr returns the ClickHouse expression that turns column into its pattern; otel.logs stores its hash as PatternHash.
func PatternExpr(column string) string {
	expr := column
	for _, r := range patternRules {
		expr = "replaceRegexpAll(" + expr + ", " + sqlString(r[0]) + ", " + sqlString(r[1]) + ")"
	}
	return "substringUTF8(" + expr + ", 1, 200)"
}

// sqlString quotes a constant as a ClickHouse string literal.
func sqlString(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}
