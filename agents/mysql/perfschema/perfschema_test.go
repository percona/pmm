// pmm-agent
// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package perfschema

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-agent/utils/tests"
)

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
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Queryid:     "Normal",
				Fingerprint: "SELECT 'Normal'",
				AgentType:   inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
				NumQueries:  5,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{
				MRowsAffectedCnt: 5,
				MRowsAffectedSum: 10, // 60-50
			},
		}
		tests.AssertBucketsEqual(t, expected, actual[0])
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
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Queryid:     "New",
				Fingerprint: "SELECT 'New'",
				AgentType:   inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
				NumQueries:  10,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{
				MRowsAffectedCnt: 10,
				MRowsAffectedSum: 50,
			},
		}
		tests.AssertBucketsEqual(t, expected, actual[0])
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
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Queryid:     "TruncateAndNew",
				Fingerprint: "SELECT 'TruncateAndNew'",
				AgentType:   inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
				NumQueries:  5,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{
				MRowsAffectedCnt: 5,
				MRowsAffectedSum: 25,
			},
		}
		tests.AssertBucketsEqual(t, expected, actual[0])
	})
}

func setup(t *testing.T, db *reform.DB) *PerfSchema {
	t.Helper()

	truncateQuery := fmt.Sprintf("TRUNCATE /* %s */ ", queryTag) //nolint:gosec
	_, err := db.Exec(truncateQuery + "performance_schema.events_statements_history")
	require.NoError(t, err)
	_, err = db.Exec(truncateQuery + "performance_schema.events_statements_summary_by_digest")
	require.NoError(t, err)

	p := newPerfSchema(db.WithTag(queryTag), nil, "agent_id", logrus.WithField("test", t.Name()))
	require.NoError(t, p.refreshHistoryCache())
	return p
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*agentpb.MetricsBucket) []*agentpb.MetricsBucket {
	res := make([]*agentpb.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.Contains(b.Common.Example, "/* pmm-agent:perfschema */"):
			continue
		case strings.Contains(b.Common.Example, "/* pmm-agent:slowlog */"):
			continue
		case strings.Contains(b.Common.Example, "/* pmm-agent:connectionchecker */"):
			continue

		case strings.Contains(b.Common.Example, "/* pmm-agent-tests:MySQLVersion */"):
			continue
		case strings.Contains(b.Common.Example, "/* pmm-agent-tests:waitForFixtures */"):
			continue
		}

		switch {
		case b.Common.Fingerprint == "ANALYZE TABLE `city`": // OpenTestMySQL
			continue
		case b.Common.Fingerprint == "SHOW GLOBAL VARIABLES WHERE `Variable_name` = ?": // MySQLVersion
			continue
		case b.Common.Fingerprint == "SELECT `id` FROM `city` LIMIT ?": // waitForFixtures
			continue
		case b.Common.Fingerprint == "SELECT ID FROM `city` LIMIT ?": // waitForFixtures for MariaDB
			continue
		case b.Common.Fingerprint == "SELECT COUNT ( * ) FROM `city`": // actions tests
			continue
		case b.Common.Fingerprint == "CREATE TABLE IF NOT EXISTS `t1` ( `col1` CHARACTER (?) ) CHARACTER SET `utf8mb4` COLLATE `utf8mb4_general_ci`": // tests for invalid characters
			continue

		case strings.HasPrefix(b.Common.Fingerprint, "SELECT @@`slow_query_log"): // slowlog
			continue
		}

		res = append(res, b)
	}
	return res
}

func TestPerfSchema(t *testing.T) {
	sqlDB := tests.OpenTestMySQL(t)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf))

	updateQuery := fmt.Sprintf("UPDATE /* %s */ ", queryTag) //nolint:gosec
	_, err := db.Exec(updateQuery + "performance_schema.setup_consumers SET ENABLED='YES' WHERE NAME='events_statements_history'")
	require.NoError(t, err, "failed to enable events_statements_history consumer")

	structs, err := db.SelectAllFrom(setupConsumersView, "ORDER BY NAME")
	require.NoError(t, err)
	tests.LogTable(t, structs)
	structs, err = db.SelectAllFrom(setupInstrumentsView, "ORDER BY NAME")
	require.NoError(t, err)
	tests.LogTable(t, structs)

	var rowsExamined float32
	mySQLVersion, mySQLVendor := tests.MySQLVersion(t, sqlDB)
	var digests map[string]string // digest_text/fingerprint to digest/query_id
	switch fmt.Sprintf("%s-%s", mySQLVersion, mySQLVendor) {
	case "5.6-oracle":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "192ad18c482d389f36ebb0aa58311236",
			"SELECT * FROM `city`": "cf5d7abca54943b1aa9e126c85a7d020",
		}
	case "5.7-oracle":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "52f680b0d3b57c2fa381f52038754db4",
			"SELECT * FROM `city`": "05292e6e5fb868ce2864918d5e934cb3",
		}

	case "5.6-percona":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "d8dc769e3126abd5578679f520bad1a5",
			"SELECT * FROM `city`": "6d3c8e264bfdd0ce5d3c81d481148a9c",
		}
	case "5.7-percona":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "049a1b20acee144f86b9a1e4aca398d6",
			"SELECT * FROM `city`": "9c799bdb2460f79b3423b77cd10403da",
		}

	case "8.0-oracle", "8.0-percona":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "0b1b1c39d4ee2dda7df2a532d0a23406d86bd34e2cd7f22e3f7e9dedadff9b69",
			"SELECT * FROM `city`": "950bdc225cf73c9096ba499351ed4376f4526abad3d8ceabc168b6b28cfc9eab",
		}
		rowsExamined = 1

	case "10.2-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "e58c348e4947db23b7f3ad30b7ed184a",
			"SELECT * FROM `city`": "e0f47172152e8750d070a854e607123f",
		}

	case "10.3-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "af50128de9089f71d749eda5ba3d02cd",
			"SELECT * FROM `city`": "2153d686f335a2ca39f3aca05bf9709a",
		}

	case "10.4-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "84a33aa2dff8b023bfd9c28247516e55",
			"SELECT * FROM `city`": "639b3ffc239a110c57ade746773952ab",
		}

	default:
		t.Log("Unhandled version, assuming dummy digests.")
		digests = map[string]string{
			"SELECT `sleep` (?)":   "TODO-sleep",
			"SELECT * FROM `city`": "TODO-star",
		}
	}

	t.Run("Sleep", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("SELECT /* Sleep */ sleep(0.1)")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0.1, actual.Common.MQueryTimeSum, 0.09)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         "SELECT `sleep` (?)",
				Schema:              "world",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:             "SELECT /* Sleep */ sleep(0.1)",
				ExampleFormat:       agentpb.ExampleFormat_EXAMPLE,
				ExampleType:         agentpb.ExampleType_RANDOM,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{
				MRowsSentCnt:     1,
				MRowsSentSum:     1,
				MRowsExaminedCnt: rowsExamined,
				MRowsExaminedSum: rowsExamined,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
	})

	t.Run("AllCities", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("SELECT /* AllCities */ * FROM city")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.Mysql.MLockTimeSum, 0.09)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         "SELECT * FROM `city`",
				Schema:              "world",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:             "SELECT /* AllCities */ * FROM city",
				ExampleFormat:       agentpb.ExampleFormat_EXAMPLE,
				ExampleType:         agentpb.ExampleType_RANDOM,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{
				MLockTimeCnt:     1,
				MLockTimeSum:     actual.Mysql.MLockTimeSum,
				MRowsSentCnt:     1,
				MRowsSentSum:     4079,
				MRowsExaminedCnt: 1,
				MRowsExaminedSum: 4079,
				MFullScanCnt:     1,
				MFullScanSum:     1,
				MNoIndexUsedCnt:  1,
				MNoIndexUsedSum:  1,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
	})

	t.Run("Invalid UTF-8", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec("CREATE TABLE if not exists t1(col1 CHAR(100)) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci")
		require.NoError(t, err)
		defer func() {
			_, err := db.Exec("DROP TABLE t1")
			require.NoError(t, err)
		}()

		_, err = db.Exec("SELECT /* t1 */ * FROM t1 where col1='Bu\xf1rk'")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())
		var example string
		switch mySQLVersion {
		// Perf schema truncates queries with non-utf8 characters.
		case "8.0":
			example = "SELECT /* t1 */ * FROM t1 where col1='Bu"
		default:
			example = "SELECT /* t1 */ * FROM t1 where col1=..."
		}

		var numQueriesWithWarnings float32
		if mySQLVendor != "mariadb" {
			numQueriesWithWarnings = 1
		}

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.Mysql.MLockTimeSum, 0.09)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:            "SELECT * FROM `t1` WHERE `col1` = ?",
				Schema:                 "world",
				AgentId:                "agent_id",
				PeriodStartUnixSecs:    1554116340,
				PeriodLengthSecs:       60,
				AgentType:              inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:                example,
				ExampleFormat:          agentpb.ExampleFormat_EXAMPLE,
				ExampleType:            agentpb.ExampleType_RANDOM,
				NumQueries:             1,
				NumQueriesWithWarnings: numQueriesWithWarnings,
				MQueryTimeCnt:          1,
				MQueryTimeSum:          actual.Common.MQueryTimeSum,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{
				MLockTimeCnt:    1,
				MLockTimeSum:    actual.Mysql.MLockTimeSum,
				MFullScanCnt:    1,
				MFullScanSum:    1,
				MNoIndexUsedCnt: 1,
				MNoIndexUsedSum: 1,
			},
		}
		// We are not testing query id here.
		actual.Common.Queryid = expected.Common.Queryid
		tests.AssertBucketsEqual(t, expected, actual)

		structs, err = db.SelectAllFrom(eventsStatementsHistoryView, "ORDER BY SQL_TEXT")
		require.NoError(t, err)
		tests.LogTable(t, structs)
	})
}
