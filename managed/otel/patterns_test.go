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

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPatternRules applies the masking rules with Go's RE2, the engine of ClickHouse's replaceRegexpAll.
func TestPatternRules(t *testing.T) {
	mask := func(s string) string {
		for _, r := range patternRules {
			s = regexp.MustCompile(r[0]).ReplaceAllLiteralString(s, r[1])
		}
		return s
	}
	for in, want := range map[string]string{
		"connection authenticated: identity='pmm' method=md5 (/etc/postgresql/17/main/pg_hba.conf:4)": "connection authenticated: identity='?' method=md5 (/etc/postgresql/?/main/pg_hba.conf:?)",
		"process 48211 still waiting for ShareLock on transaction 9912044 after 1000.087 ms":          "process ? still waiting for ShareLock on transaction ? after ? ms",
		"execute <unnamed>: SELECT pg_database_size($1)":                                              "execute <unnamed>: SELECT pg_database_size($?)",
		"connection received: host=10.30.51.101 port=53068":                                           "connection received: host=? port=?",
		"Applied op on rs103 shard":                                                                   "Applied op on rs103 shard",
	} {
		assert.Equal(t, want, mask(in))
	}
}

func TestPatternExpr(t *testing.T) {
	expr := PatternExpr("Body")
	assert.True(t, strings.HasPrefix(expr, "substringUTF8(replaceRegexpAll("), expr)
	assert.True(t, strings.HasSuffix(expr, ", 1, 200)"), expr)
}
