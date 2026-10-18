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
	backref := regexp.MustCompile(`\\([0-9])`)
	mask := func(s string) string {
		for _, r := range patternRules {
			s = regexp.MustCompile(r[0]).ReplaceAllString(s, backref.ReplaceAllString(r[1], `$${$1}`))
		}
		return s
	}
	for in, want := range map[string]string{
		"connection authenticated: identity='pmm' method=md5 (/etc/postgresql/17/main/pg_hba.conf:4)": "connection authenticated: identity='?' method=md5 (/etc/postgresql/?/main/pg_hba.conf:?)",
		"process 48211 still waiting for ShareLock on transaction 9912044 after 1000.087 ms":          "process ? still waiting for ShareLock on transaction ? after ? ms",
		"execute <unnamed>: SELECT pg_database_size($1)":                                              "execute <unnamed>: SELECT pg_database_size($?)",
		"connection received: host=10.30.51.101 port=53068":                                           "connection received: host=? port=?",
		"Applied op on rs103 shard":                                                                   "Applied op on rs103 shard",
		"utf8mb4 on x86_64 with v2 and sha256_password":                                               "utf8mb4 on x86_64 with v2 and sha256_password",

		"statement: SELECT * FROM t WHERE id IN (1, 2, 3) AND s IN ('a','b')": "statement: SELECT * FROM t WHERE id IN (?) AND s IN ('?')",
		"Page cleaner took 4048ms to flush 8 and evict 0 pages":               "Page cleaner took ? to flush ? and evict ? pages",

		"otel.logs (9e93d6ee-1b8a-4873-bc5f-54eba47a42e2) (MergerMutator): Merged 5 parts: " +
			"[20261018_207652_207740_22, 20261018_207744_207744_0] -> 20261018_207652_207744_23": "otel.logs (?) (MergerMutator): Merged ? parts: [?] -> ?",
		"otel.logs (9e93d6ee-1b8a-4873-bc5f-54eba47a42e2) (MergerMutator): Selected 6 parts from " +
			"20261018_204744_204744_0 to 20261018_204749_204749_0. Merge selecting phase took: 0ms": "otel.logs (?) (MergerMutator): Selected ? parts from ? to ?. Merge selecting phase took: ?",

		"2026/10/18 17:54:56 [warn] 93144#93144: *707761 a client request body is buffered": "? [warn] ?#?: *? a client request body is buffered",

		`logger=dashboard-service t=2026-10-18T19:24:14.490949962Z level=info msg="No last resource version found" orgID=1`: "logger=dashboard-service t=? level=info msg=No last resource version found orgID=?",
		`time="2026-10-18T17:55:00.588+00:00" level=info msg="Saved 184 buckets in 567.33747ms." component=data_ingestion`:  "time='?' level=info msg=Saved ? buckets in ?. component=data_ingestion",
		`time="2026-10-18T17:54:47.422+00:00" level=info msg="time=2026-10-18T17:54:47.421Z level=INFO source=main.go:112 ` +
			`msg=\"Excluded databases\" databases=\"[template0 template1]\"" agentID=87ed8ca0-ea07-4484-8025-7da9ae86c972`: `time='?' level=info msg=time=? level=INFO source=main.go:? msg=Excluded databases databases=\'?' agentID=?`,
	} {
		assert.Equal(t, want, mask(in), in)
	}
}

func TestPatternExpr(t *testing.T) {
	expr := PatternExpr("Body")
	assert.True(t, strings.HasPrefix(expr, "substringUTF8(replaceRegexpAll("), expr)
	assert.True(t, strings.HasSuffix(expr, ", 1, 200)"), expr)
	assert.Contains(t, expr, `'msg=\\1'`, "a back-reference reaches ClickHouse as \\1")
}
