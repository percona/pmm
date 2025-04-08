// Copyright (C) 2023 Percona LLC
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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/truncate"
	"github.com/percona/pmm/agent/utils/version"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestPerfSchemaMakeBuckets(t *testing.T) {
	defaultMaxQueryLength := truncate.GetDefaultMaxQueryLength()
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
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()), defaultMaxQueryLength)
		require.Len(t, actual, 1)
		expected := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid:     "Normal",
				Fingerprint: "SELECT 'Normal'",
				AgentType:   inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				NumQueries:  5,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{
				MRowsAffectedCnt: 5,
				MRowsAffectedSum: 10, // 60-50
			},
		}
		tests.AssertBucketsEqual(t, expected, actual[0])
	})

	t.Run("New", func(t *testing.T) {
		prev := make(map[string]*eventsStatementsSummaryByDigest)
		current := map[string]*eventsStatementsSummaryByDigest{
			"New": {
				Digest:          pointer.ToString("New"),
				DigestText:      pointer.ToString("SELECT 'New'"),
				CountStar:       10,
				SumRowsAffected: 50,
			},
		}
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()), defaultMaxQueryLength)
		require.Len(t, actual, 1)
		expected := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid:     "New",
				Fingerprint: "SELECT 'New'",
				AgentType:   inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				NumQueries:  10,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{
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
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()), defaultMaxQueryLength)
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
		current := make(map[string]*eventsStatementsSummaryByDigest)
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()), defaultMaxQueryLength)
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
		actual := makeBuckets(current, prev, logrus.WithField("test", t.Name()), defaultMaxQueryLength)
		require.Len(t, actual, 1)
		expected := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid:     "TruncateAndNew",
				Fingerprint: "SELECT 'TruncateAndNew'",
				AgentType:   inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				NumQueries:  5,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{
				MRowsAffectedCnt: 5,
				MRowsAffectedSum: 25,
			},
		}
		tests.AssertBucketsEqual(t, expected, actual[0])
	})
}

type setupParams struct {
	db                   *reform.DB
	maxQueryLength       int32
	disableQueryExamples bool
}

func setup(t *testing.T, sp *setupParams) *PerfSchema {
	t.Helper()

	truncateQuery := fmt.Sprintf("TRUNCATE /* %s */ ", queryTag)
	_, err := sp.db.Exec(truncateQuery + "performance_schema.events_statements_history")
	require.NoError(t, err)
	_, err = sp.db.Exec(truncateQuery + "performance_schema.events_statements_summary_by_digest")
	require.NoError(t, err)

	newParams := &newPerfSchemaParams{
		Querier:              sp.db.WithTag(queryTag),
		DBCloser:             nil,
		AgentID:              "agent_id",
		MaxQueryLength:       sp.maxQueryLength,
		DisableQueryExamples: sp.disableQueryExamples,
		LogEntry:             logrus.WithField("test", t.Name()),
	}

	p, err := newPerfSchema(newParams)
	require.NoError(t, err)
	require.NoError(t, p.refreshHistoryCache())
	return p
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*agentv1.MetricsBucket) []*agentv1.MetricsBucket {
	filterList := map[string]struct{}{
		"ANALYZE TABLE `city`":                            {}, // OpenTestMySQL
		"SHOW GLOBAL VARIABLES WHERE `Variable_name` = ?": {}, // MySQLVersion
		"SHOW VARIABLES LIKE ?":                           {}, // MariaDBVersion
		"SELECT `id` FROM `city` LIMIT ?":                 {}, // waitForFixtures
		"SELECT ID FROM `city` LIMIT ?":                   {}, // waitForFixtures for MariaDB
		"SELECT COUNT ( * ) FROM `city`":                  {}, // actions tests
		"CREATE TABLE IF NOT EXISTS `t1` ( `col1` CHARACTER (?) ) CHARACTER SET `utf8mb4` COLLATE `utf8mb4_general_ci`": {}, // tests for invalid characters
	}
	res := make([]*agentv1.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.Contains(b.Common.Example, "/* agent='perfschema' */"):
			continue
		case strings.Contains(b.Common.Example, "/* agent='slowlog' */"):
			continue
		case strings.Contains(b.Common.Example, "/* agent='connectionchecker' */"):
			continue

		case strings.Contains(b.Common.Example, "/* pmm-agent-tests:MySQLVersion */"):
			continue
		case strings.Contains(b.Common.Example, "/* pmm-agent-tests:waitForFixtures */"):
			continue
		case strings.Contains(b.Common.Fingerprint, "events_statements_history"):
			continue
		case strings.HasPrefix(b.Common.Fingerprint, "SELECT @@`slow_query_log"): // slowlog
			continue
		case strings.HasPrefix(b.Common.Fingerprint, "TRUNCATE"): // OpenTestMySQL
			continue
		}
		if _, ok := filterList[b.Common.Fingerprint]; ok {
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

	updateQuery := fmt.Sprintf("UPDATE /* %s */ ", queryTag)
	_, err := db.Exec(updateQuery + "performance_schema.setup_consumers SET ENABLED='YES'")
	require.NoError(t, err, "failed to enable events_statements_history consumer")

	structs, err := db.SelectAllFrom(setupConsumersView, "ORDER BY NAME")
	require.NoError(t, err)
	tests.LogTable(t, structs)
	structs, err = db.SelectAllFrom(setupInstrumentsView, "ORDER BY NAME")
	require.NoError(t, err)
	tests.LogTable(t, structs)

	var rowsExamined float32
	ctx := context.Background()
	mySQLVersion, mySQLVendor, _ := version.GetMySQLVersion(ctx, db.WithTag("pmm-agent-tests:MySQLVersion"))
	t.Logf("MySQL version: %s, vendor: %s", mySQLVersion, mySQLVendor)
	var digests map[string]string // digest_text/fingerprint to digest/query_id
	switch fmt.Sprintf("%s-%s", mySQLVersion, mySQLVendor) {
	case "5.6-oracle":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-192ad18c482d389f36ebb0aa58311236",
			"SELECT * FROM `city`": "world-cf5d7abca54943b1aa9e126c85a7d020",
		}
	case "5.7-oracle":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-52f680b0d3b57c2fa381f52038754db4",
			"SELECT * FROM `city`": "world-05292e6e5fb868ce2864918d5e934cb3",
		}

	case "5.6-percona":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-d8dc769e3126abd5578679f520bad1a5",
			"SELECT * FROM `city`": "world-6d3c8e264bfdd0ce5d3c81d481148a9c",
		}
	case "5.7-percona":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-049a1b20acee144f86b9a1e4aca398d6",
			"SELECT * FROM `city`": "world-9c799bdb2460f79b3423b77cd10403da",
		}

	case "8.0-oracle", "8.0-percona", "8.4-oracle", "9.0-oracle", "9.1-oracle", "9.2-oracle":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-0b1b1c39d4ee2dda7df2a532d0a23406d86bd34e2cd7f22e3f7e9dedadff9b69",
			"SELECT * FROM `city`": "world-950bdc225cf73c9096ba499351ed4376f4526abad3d8ceabc168b6b28cfc9eab",
		}
		rowsExamined = 1

	case "10.2-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-fe8d67e28d171893e1b33b179394e592",
			"SELECT * FROM `city`": "world-7e30fa1763d6d9aa88f359236cedaa78",
		}

	case "10.3-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-b0062e3bc75dd6e57cdc90696ba47688",
			"SELECT * FROM `city`": "world-f4c92872bdf2de2331aae63a94b51a83",
		}

	case "10.4-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-0a01e0e8325cdd1db9a0746270ab8ce9",
			"SELECT * FROM `city`": "world-a65e76b1643273fa3206b11c4f4d8739",
		}

	case "11.2-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-ffbde6c4dfda8dff9a4fefd7e8ed648f",
			"SELECT * FROM `city`": "world-d0f2ac0577a44d383c5c0480a420caeb",
		}

	case "11.4-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-860792b8f3d058489b287e30ccf3beae",
			"SELECT * FROM `city`": "world-457a868ea48e4571327914f2831d62f5",
		}

	case "11.5-mariadb":
		digests = map[string]string{
			"SELECT `sleep` (?)":   "world-860792b8f3d058489b287e30ccf3beae",
			"SELECT * FROM `city`": "world-457a868ea48e4571327914f2831d62f5",
		}

	default:
		t.Logf("Unhandled version, assuming dummy digests. MySQL version: %s, vendor: %s", mySQLVersion, mySQLVendor)
		digests = map[string]string{
			"SELECT `sleep` (?)":   "TODO-sleep",
			"SELECT * FROM `city`": "TODO-star",
		}
	}

	t.Run("Sleep", func(t *testing.T) {
		m := setup(t, &setupParams{
			db:                   db,
			disableQueryExamples: false,
		})

		_, err := db.Exec("SELECT /* Sleep controller='test' */ sleep(0.1)")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0.1, actual.Common.MQueryTimeSum, 0.09)

		expected := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				ExplainFingerprint:  "SELECT `sleep` (:1)",
				PlaceholdersCount:   1,
				Comments:            map[string]string{"controller": "test"},
				Fingerprint:         "SELECT `sleep` (?)",
				Schema:              "world",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:             "SELECT /* Sleep controller='test' */ sleep(0.1)",
				ExampleType:         agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{
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
		m := setup(t, &setupParams{
			db:                   db,
			disableQueryExamples: false,
		})

		_, err := db.Exec("SELECT /* AllCities controller='test' */ * FROM city")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.Mysql.MLockTimeSum, 0.09)
		expected := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				ExplainFingerprint:  "SELECT * FROM `city`",
				Fingerprint:         "SELECT * FROM `city`",
				Comments:            map[string]string{"controller": "test"},
				Schema:              "world",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:             "SELECT /* AllCities controller='test' */ * FROM city",
				ExampleType:         agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{
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
		m := setup(t, &setupParams{
			db:                   db,
			disableQueryExamples: false,
		})

		_, err := db.Exec("CREATE TABLE if not exists t1(col1 CHAR(100)) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci")
		require.NoError(t, err)
		defer func() {
			_, err := db.Exec("DROP TABLE t1")
			require.NoError(t, err)
		}()

		_, err = db.Exec("SELECT /* t1 controller='test' */ * FROM t1 where col1='Bu\xf1rk'")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())
		var example string
		switch {
		// Perf schema truncates queries with non-utf8 characters.
		case (mySQLVendor == version.PerconaVendor || mySQLVendor == version.OracleVendor) && mySQLVersion.Float() >= 8.0:
			example = "SELECT /* t1 controller='test' */ * FROM t1 where col1='Bu"
		default:
			example = "SELECT /* t1 controller='test' */ * FROM t1 where col1=..."
		}

		var numQueriesWithWarnings float32
		if mySQLVendor != version.MariaDBVendor {
			numQueriesWithWarnings = 1
		}

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		require.Len(t, buckets, 1, "%s", tests.FormatBuckets(buckets))

		actual := buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.Mysql.MLockTimeSum, 0.09)
		expected := &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				ExplainFingerprint:     "SELECT * FROM `t1` WHERE `col1` = :1",
				PlaceholdersCount:      1,
				Fingerprint:            "SELECT * FROM `t1` WHERE `col1` = ?",
				Comments:               map[string]string{"controller": "test"},
				Schema:                 "world",
				AgentId:                "agent_id",
				PeriodStartUnixSecs:    1554116340,
				PeriodLengthSecs:       60,
				AgentType:              inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT,
				Example:                example,
				ExampleType:            agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
				NumQueries:             1,
				NumQueriesWithWarnings: numQueriesWithWarnings,
				MQueryTimeCnt:          1,
				MQueryTimeSum:          actual.Common.MQueryTimeSum,
			},
			Mysql: &agentv1.MetricsBucket_MySQL{
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

	t.Run("DisableQueryExamples", func(t *testing.T) {
		m := setup(t, &setupParams{
			db:                   db,
			disableQueryExamples: true,
		})
		_, err = db.Exec("SELECT 1, 2, 3, 4, id FROM city WHERE id = 1")
		require.NoError(t, err)

		require.NoError(t, m.refreshHistoryCache())

		buckets, err := m.getNewBuckets(time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)

		require.NotEmpty(t, buckets)
		for _, b := range buckets {
			assert.NotEmpty(t, b.Common.Queryid)
			assert.NotEmpty(t, b.Common.Fingerprint)
			assert.Empty(t, b.Common.Example)
		}
	})
}
