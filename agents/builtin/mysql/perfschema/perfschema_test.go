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

package perfschema

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-agent/utils/tests"
)

func assertBucketsEqual(t *testing.T, expected, actual *qanpb.MetricsBucket) bool {
	t.Helper()
	return assert.Equal(t, proto.MarshalTextString(expected), proto.MarshalTextString(actual))
}

func TestPerfSchemaMakeBuckets(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		prev := map[string]*eventsStatementsSummaryByDigest{
			"Normal": {
				Digest:          pointer.ToString("Normal"),
				DigestText:      pointer.ToString("SELECT 'Normal'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		current := map[string]*eventsStatementsSummaryByDigest{
			"Normal": {
				Digest:          pointer.ToString("Normal"),
				DigestText:      pointer.ToString("SELECT 'Normal'"),
				CountStar:       15, // +5
				SumRowsAffected: 60, // +10
			},
		}
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()))
		require.Len(t, actual, 1)
		expected := &qanpb.MetricsBucket{
			Queryid:          "Normal",
			Fingerprint:      "SELECT 'Normal'",
			MetricsSource:    qanpb.MetricsSource_MYSQL_PERFSCHEMA,
			NumQueries:       5,
			MRowsAffectedCnt: 5,
			MRowsAffectedSum: 10, // 60-50
		}
		assertBucketsEqual(t, expected, actual[0])
	})

	t.Run("New", func(t *testing.T) {
		prev := map[string]*eventsStatementsSummaryByDigest{}
		current := map[string]*eventsStatementsSummaryByDigest{
			"New": {
				Digest:          pointer.ToString("New"),
				DigestText:      pointer.ToString("SELECT 'New'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()))
		require.Len(t, actual, 1)
		expected := &qanpb.MetricsBucket{
			Queryid:          "New",
			Fingerprint:      "SELECT 'New'",
			MetricsSource:    qanpb.MetricsSource_MYSQL_PERFSCHEMA,
			NumQueries:       10,
			MRowsAffectedCnt: 10,
			MRowsAffectedSum: 50,
		}
		assertBucketsEqual(t, expected, actual[0])
	})

	t.Run("Same", func(t *testing.T) {
		prev := map[string]*eventsStatementsSummaryByDigest{
			"Same": {
				Digest:          pointer.ToString("Same"),
				DigestText:      pointer.ToString("SELECT 'Same'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		current := map[string]*eventsStatementsSummaryByDigest{
			"Same": {
				Digest:          pointer.ToString("Same"),
				DigestText:      pointer.ToString("SELECT 'Same'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()))
		require.Len(t, actual, 0)
	})

	t.Run("Truncate", func(t *testing.T) {
		prev := map[string]*eventsStatementsSummaryByDigest{
			"Truncate": {
				Digest:          pointer.ToString("Truncate"),
				DigestText:      pointer.ToString("SELECT 'Truncate'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		current := map[string]*eventsStatementsSummaryByDigest{}
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()))
		require.Len(t, actual, 0)
	})

	t.Run("TruncateAndNew", func(t *testing.T) {
		prev := map[string]*eventsStatementsSummaryByDigest{
			"TruncateAndNew": {
				Digest:          pointer.ToString("TruncateAndNew"),
				DigestText:      pointer.ToString("SELECT 'TruncateAndNew'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		current := map[string]*eventsStatementsSummaryByDigest{
			"TruncateAndNew": {
				Digest:          pointer.ToString("TruncateAndNew"),
				DigestText:      pointer.ToString("SELECT 'TruncateAndNew'"),
				CountStar:       5,
				SumRowsAffected: 25,
			},
		}
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()))
		require.Len(t, actual, 1)
		expected := &qanpb.MetricsBucket{
			Queryid:          "TruncateAndNew",
			Fingerprint:      "SELECT 'TruncateAndNew'",
			MetricsSource:    qanpb.MetricsSource_MYSQL_PERFSCHEMA,
			NumQueries:       5,
			MRowsAffectedCnt: 5,
			MRowsAffectedSum: 25,
		}
		assertBucketsEqual(t, expected, actual[0])
	})
}

func logTable(t *testing.T, structs []reform.Struct) {
	t.Helper()

	if len(structs) == 0 {
		t.Log("logTable: empty")
		return
	}

	columns := structs[0].View().Columns()
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	_, err := fmt.Fprintln(w, strings.Join(columns, "\t"))
	require.NoError(t, err)
	for i, c := range columns {
		columns[i] = strings.Repeat("-", len(c))
	}
	_, err = fmt.Fprintln(w, strings.Join(columns, "\t"))
	require.NoError(t, err)

	for _, str := range structs {
		res := make([]string, len(str.Values()))
		for i, v := range str.Values() {
			res[i] = spew.Sprint(v)
		}
		fmt.Fprintf(w, "%s\n", strings.Join(res, "\t"))
	}

	require.NoError(t, w.Flush())
	t.Logf("%s:\n%s", structs[0].View().Name(), buf.Bytes())
}

func setup(t *testing.T, db *reform.DB) *PerfSchema {
	t.Helper()

	_, err := db.Exec("TRUNCATE performance_schema.events_statements_history")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE performance_schema.events_statements_summary_by_digest")
	require.NoError(t, err)

	return newPerfSchema(db, "agent_id", logrus.WithField("test", t.Name()))
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*qanpb.MetricsBucket) []*qanpb.MetricsBucket {
	res := make([]*qanpb.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.HasPrefix(b.Fingerprint, "SELECT @@`skip_networking`"):
			continue

		case strings.HasPrefix(b.Fingerprint, "TRUNCATE `performance_schema`"):
			continue
		case strings.HasPrefix(b.Fingerprint, "SELECT `performance_schema`"):
			continue

		default:
			res = append(res, b)
		}
	}
	return res
}

func TestPerfSchema(t *testing.T) {
	sqlDB := tests.OpenTestMySQL(t)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))

	_, err := db.Exec("UPDATE performance_schema.setup_consumers SET ENABLED='YES' WHERE NAME='events_statements_history'")
	require.NoError(t, err, "failed to enable events_statements_history consumer")

	structs, err := db.SelectAllFrom(setupConsumersView, "ORDER BY NAME")
	require.NoError(t, err)
	logTable(t, structs)
	structs, err = db.SelectAllFrom(setupInstrumentsView, "ORDER BY NAME")
	require.NoError(t, err)
	logTable(t, structs)

	var digests map[string]string // digest_text/fingerprint to digest/query_id
	switch tests.MySQLVersion(t, sqlDB) {
	case "5.6":
		digests = map[string]string{
			"SELECT ?":             "41782b6b3af16c6426fb64b88a51d8a5",
			"SELECT `sleep` (?)":   "d8dc769e3126abd5578679f520bad1a5",
			"SELECT * FROM `city`": "6d3c8e264bfdd0ce5d3c81d481148a9c",
		}
	case "5.7":
		digests = map[string]string{
			"SELECT ?":             "3fff4c5a5ca5e1e484663cab257efd1e",
			"SELECT `sleep` (?)":   "049a1b20acee144f86b9a1e4aca398d6",
			"SELECT * FROM `city`": "9c799bdb2460f79b3423b77cd10403da",
		}
	case "8.0":
		digests = map[string]string{
			"SELECT ?":             "d1b44b0c19af710b5a679907e284acd2ddc285201794bc69a2389d77baedddae",
			"SELECT `sleep` (?)":   "0b1b1c39d4ee2dda7df2a532d0a23406d86bd34e2cd7f22e3f7e9dedadff9b69",
			"SELECT * FROM `city`": "950bdc225cf73c9096ba499351ed4376f4526abad3d8ceabc168b6b28cfc9eab",
		}
	default:
		t.Log("Unhandled version, assuming dummy digests.")
		digests = map[string]string{
			"SELECT ?":             "TODO",
			"SELECT `sleep` (?)":   "TODO",
			"SELECT * FROM `city`": "TODO",
		}
	}

	t.Run("Sleep", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("SELECT /* Sleep */ sleep(0.1)")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60*time.Second)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1)

		actual := buckets[0]
		assert.InDelta(t, 0.1, actual.MQueryTimeSum, 0.09)
		expected := &qanpb.MetricsBucket{
			Fingerprint:         "SELECT `sleep` (?)",
			DSchema:             "world",
			AgentId:             "agent_id",
			PeriodStartUnixSecs: 1554116340,
			PeriodLengthSecs:    60,
			MetricsSource:       qanpb.MetricsSource_MYSQL_PERFSCHEMA,
			Example:             "SELECT /* Sleep */ sleep(0.1)",
			ExampleFormat:       qanpb.ExampleFormat_EXAMPLE,
			ExampleType:         qanpb.ExampleType_RANDOM,
			NumQueries:          1,
			MQueryTimeCnt:       1,
			MQueryTimeSum:       actual.MQueryTimeSum,
			MRowsSentCnt:        1,
			MRowsSentSum:        1,
		}
		expected.Queryid = digests[expected.Fingerprint]
		assertBucketsEqual(t, expected, actual)
	})

	t.Run("AllCities", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("SELECT /* AllCities */ * FROM city")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60*time.Second)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1)

		actual := buckets[0]
		assert.InDelta(t, 0, actual.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.MLockTimeSum, 0.09)
		expected := &qanpb.MetricsBucket{
			Fingerprint:         "SELECT * FROM `city`",
			DSchema:             "world",
			AgentId:             "agent_id",
			PeriodStartUnixSecs: 1554116340,
			PeriodLengthSecs:    60,
			MetricsSource:       qanpb.MetricsSource_MYSQL_PERFSCHEMA,
			Example:             "SELECT /* AllCities */ * FROM city",
			ExampleFormat:       qanpb.ExampleFormat_EXAMPLE,
			ExampleType:         qanpb.ExampleType_RANDOM,
			NumQueries:          1,
			MQueryTimeCnt:       1,
			MQueryTimeSum:       actual.MQueryTimeSum,
			MLockTimeCnt:        1,
			MLockTimeSum:        actual.MLockTimeSum,
			MRowsSentCnt:        1,
			MRowsSentSum:        4079,
			MRowsExaminedCnt:    1,
			MRowsExaminedSum:    4079,
			MFullScanCnt:        1,
			MFullScanSum:        1,
			MNoIndexUsedCnt:     1,
			MNoIndexUsedSum:     1,
		}
		expected.Queryid = digests[expected.Fingerprint]
		assertBucketsEqual(t, expected, actual)
	})
}
