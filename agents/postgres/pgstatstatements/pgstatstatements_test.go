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

package pgstatstatements

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-agent/utils/tests"
)

func setup(t *testing.T, db *reform.DB) *PGStatStatementsQAN {
	t.Helper()

	selectQuery := fmt.Sprintf("SELECT /* %s */ ", queryTag) //nolint:gosec

	_, err := db.Exec(selectQuery + "pg_stat_statements_reset()")
	require.NoError(t, err)

	return newPgStatStatementsQAN(db.WithTag(queryTag), nil, "agent_id", logrus.WithField("test", t.Name()))
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*qanpb.MetricsBucket) []*qanpb.MetricsBucket {
	res := make([]*qanpb.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.Contains(b.Fingerprint, "/* pmm-agent:pgstatstatements */"):
			continue
		case strings.Contains(b.Fingerprint, "/* pmm-agent:connectionchecker */"):
			continue

		case strings.Contains(b.Fingerprint, "/* pmm-agent-tests:PostgreSQLVersion */"):
			continue

		default:
			res = append(res, b)
		}
	}
	return res
}

func TestPGStatStatementsQAN(t *testing.T) {
	sqlDB := tests.OpenTestPostgreSQL(t)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public")
	require.NoError(t, err)

	structs, err := db.SelectAllFrom(pgStatDatabaseView, "")
	require.NoError(t, err)
	tests.LogTable(t, structs)
	structs, err = db.SelectAllFrom(pgStatStatementsView, "")
	require.NoError(t, err)
	tests.LogTable(t, structs)

	engineVersion := tests.PostgreSQLVersion(t, sqlDB)
	var digests map[string]string // digest_text/fingerprint to digest/query_id
	switch engineVersion {
	case "9.4":
		digests = map[string]string{
			"SELECT /* AllCities */ * FROM city": "3239586867",
		}
	case "9.5", "9.6":
		digests = map[string]string{
			"SELECT /* AllCities */ * FROM city": "3994135135",
		}
	case "10":
		digests = map[string]string{
			"SELECT /* AllCities */ * FROM city": "2229807896",
		}
	case "11":
		digests = map[string]string{
			"SELECT /* AllCities */ * FROM city": "-4056421706168012289",
		}
	case "12":
		digests = map[string]string{
			"SELECT /* AllCities */ * FROM city": "5627444073676588515",
		}

	default:
		t.Log("Unhandled version, assuming dummy digests.")
		digests = map[string]string{
			"SELECT /* AllCities */ * FROM city": "TODO-star",
		}
	}

	t.Run("AllCities", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("SELECT /* AllCities */ * FROM city")
		require.NoError(t, err)

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0, actual.MQueryTimeSum, 0.09)
		assert.Equal(t, float32(33), actual.MSharedBlksHitSum+actual.MSharedBlksReadSum)
		assert.InDelta(t, 1.5, actual.MSharedBlksHitCnt+actual.MSharedBlksReadCnt, 0.5)
		expected := &qanpb.MetricsBucket{
			Fingerprint:         "SELECT /* AllCities */ * FROM city",
			Schema:              "pmm-agent",
			AgentId:             "agent_id",
			PeriodStartUnixSecs: 1554116340,
			PeriodLengthSecs:    60,
			AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
			NumQueries:          1,
			MQueryTimeCnt:       1,
			MQueryTimeSum:       actual.MQueryTimeSum,
			MSharedBlksReadCnt:  actual.MSharedBlksReadCnt,
			MSharedBlksReadSum:  actual.MSharedBlksReadSum,
			MSharedBlksHitCnt:   actual.MSharedBlksHitCnt,
			MSharedBlksHitSum:   actual.MSharedBlksHitSum,
			MRowsSentCnt:        1,
			MRowsSentSum:        4079,
		}
		expected.Queryid = digests[expected.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)

		_, err = db.Exec("SELECT /* AllCities */ * FROM city")
		require.NoError(t, err)

		buckets, err = m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual = buckets[0]
		assert.InDelta(t, 0, actual.MQueryTimeSum, 0.09)
		expected = &qanpb.MetricsBucket{
			Fingerprint:         "SELECT /* AllCities */ * FROM city",
			Schema:              "pmm-agent",
			AgentId:             "agent_id",
			PeriodStartUnixSecs: 1554116340,
			PeriodLengthSecs:    60,
			AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
			NumQueries:          1,
			MQueryTimeCnt:       1,
			MQueryTimeSum:       actual.MQueryTimeSum,
			MSharedBlksHitCnt:   1,
			MSharedBlksHitSum:   33,
			MRowsSentCnt:        1,
			MRowsSentSum:        4079,
		}
		expected.Queryid = digests[expected.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
	})
}
