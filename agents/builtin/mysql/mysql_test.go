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

package mysql

import (
	"regexp"
	"testing"

	"github.com/percona/pmm/api/qanpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-agent/utils/tests"
)

func TestGet(t *testing.T) {
	sqlDB := tests.OpenTestMySQL(t)
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))
	m := New(nil, nil)

	var version string
	err := db.QueryRow("SELECT version()").Scan(&version)
	require.NoError(t, err)
	t.Logf("version = %q", version)
	version = regexp.MustCompile(`^\d\.\d`).FindString(version)
	var digests map[string]string // digest_text/fingerprint to digest/query_id
	switch version {
	case "5.6":
		digests = map[string]string{
			"TRUNCATE `performance_schema` . `events_statements_summary_by_digest`": "3984f1508fbf01121d4cbe1b738aba23",
			"SELECT ?": "41782b6b3af16c6426fb64b88a51d8a5",
		}
	case "5.7":
		digests = map[string]string{
			"TRUNCATE `performance_schema` . `events_statements_summary_by_digest`": "0eaf45fe39b87f0be36d5e47037fc654",
			"SELECT ?": "3fff4c5a5ca5e1e484663cab257efd1e",
		}
	case "8.0":
		digests = map[string]string{
			"TRUNCATE `performance_schema` . `events_statements_summary_by_digest`": "bea5ce9985044648518884aad3633801e8446d0006f2efd6a76028555e59719f",
			"SELECT ?": "d1b44b0c19af710b5a679907e284acd2ddc285201794bc69a2389d77baedddae",
		}
	default:
		t.Fatalf("unexpected version %q", version)
	}

	_, err = db.Exec("TRUNCATE performance_schema.events_statements_summary_by_digest")
	require.NoError(t, err)

	_, err = db.Exec("SELECT 'TestGet'")
	require.NoError(t, err)

	req, err := m.get(db.Querier)
	require.NoError(t, err)
	require.Len(t, req.MetricsBucket, 2)

	actual := req.MetricsBucket[0]
	expected := &qanpb.MetricsBucket{
		Queryid:     digests[actual.Fingerprint],
		Fingerprint: "TRUNCATE `performance_schema` . `events_statements_summary_by_digest`",
		DServer:     "TODO",
		DDatabase:   "TODO",
		DSchema:     "TODO",
	}
	assert.Equal(t, expected, actual)

	actual = req.MetricsBucket[1]
	expected = &qanpb.MetricsBucket{
		Queryid:     digests[actual.Fingerprint],
		Fingerprint: "SELECT ?",
		DServer:     "TODO",
		DDatabase:   "TODO",
		DSchema:     "TODO",
	}
	assert.Equal(t, expected, actual)
}
