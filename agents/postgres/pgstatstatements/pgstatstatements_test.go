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
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-agent/utils/tests"
)

func assertBucketsEqual(t *testing.T, expected, actual *qanpb.MetricsBucket) bool {
	t.Helper()
	return assert.Equal(t, proto.MarshalTextString(expected), proto.MarshalTextString(actual))
}

func setup(t *testing.T, db *reform.DB) *PGStatStatementsQAN {
	t.Helper()

	_, err := db.Exec(`SELECT pg_stat_statements_reset()`)
	require.NoError(t, err)

	return newPgStatStatementsQAN(db, "agent_id", logrus.WithField("test", t.Name()))
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*qanpb.MetricsBucket) []*qanpb.MetricsBucket {
	res := make([]*qanpb.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.HasPrefix(b.Fingerprint, "SELECT version()"):
			continue
		case strings.HasPrefix(b.Fingerprint, "SELECT pg_stat_statements_reset()"):
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
			"SELECT * FROM city": "2500439221",
		}
	case "9.5", "9.6":
		digests = map[string]string{
			"SELECT * FROM city": "3778117319",
		}
	case "10":
		digests = map[string]string{
			"SELECT * FROM city": "952213449",
		}
	case "11":
		digests = map[string]string{
			"SELECT * FROM city": "-6046499049124467328",
		}

	default:
		t.Log("Unhandled version, assuming dummy digests.")
		digests = map[string]string{
			"SELECT * FROM city": "TODO-star",
		}
	}

	t.Run("AllCities", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("SELECT * FROM city")
		require.NoError(t, err)

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1)

		actual := buckets[0]
		assert.InDelta(t, 0, actual.MQueryTimeSum, 0.09)
		//assert.InDelta(t, 0, actual.MLockTimeSum, 0.09)
		expected := &qanpb.MetricsBucket{
			Fingerprint:         "SELECT * FROM city",
			Schema:              "pmm-agent",
			AgentId:             "agent_id",
			PeriodStartUnixSecs: 1554116340,
			PeriodLengthSecs:    60,
			AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
			//Example:             "SELECT /* AllCities */ * FROM city",
			//ExampleFormat:       qanpb.ExampleFormat_EXAMPLE,
			//ExampleType:         qanpb.ExampleType_RANDOM,
			NumQueries:    1,
			MQueryTimeCnt: 1,
			MQueryTimeSum: actual.MQueryTimeSum,
			//MLockTimeCnt:        1,
			//MLockTimeSum:        actual.MLockTimeSum,
			MRowsSentCnt: 1,
			MRowsSentSum: 4079,
			//MRowsExaminedCnt:    1,
			//MRowsExaminedSum:    4079,
			//MFullScanCnt:        1,
			//MFullScanSum:        1,
			//MNoIndexUsedCnt:     1,
			//MNoIndexUsedSum:     1,
		}
		expected.Queryid = digests[expected.Fingerprint]
		assertBucketsEqual(t, expected, actual)
	})
}
