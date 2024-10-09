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

package pgstatmonitor

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	ver "github.com/hashicorp/go-version"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/truncate"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

func setup(t *testing.T, db *reform.DB, disableCommentsParsing, disableQueryExamples bool) *PGStatMonitorQAN { //nolint:unparam
	t.Helper()

	selectQuery := fmt.Sprintf("SELECT /* %s */ ", queryTag)
	_, err := db.Exec(selectQuery + "* from pg_stat_monitor_reset()")
	require.NoError(t, err)

	pgStatMonitorQAN, err := newPgStatMonitorQAN(db.WithTag(queryTag), nil, "agent_id", disableCommentsParsing, disableQueryExamples, truncate.GetDefaultMaxQueryLength(), logrus.WithField("test", t.Name()))
	require.NoError(t, err)

	return pgStatMonitorQAN
}

func supportedVersion(version string) bool {
	supported := float64(11)
	current, err := strconv.ParseFloat(version, 32)
	if err != nil {
		return false
	}

	return current >= supported
}

func extensionExists(db *reform.DB) bool {
	var name string
	err := db.QueryRow("SELECT name FROM pg_available_extensions WHERE name='pg_stat_monitor'").Scan(&name)
	return err == nil
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*agentpb.MetricsBucket) []*agentpb.MetricsBucket {
	res := make([]*agentpb.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.Contains(b.Common.Fingerprint, "/* agent='pgstatmonitor' */"):
			continue
		case strings.Contains(b.Common.Example, "/* agent='pgstatmonitor' */"):
			continue
		case strings.Contains(b.Common.Fingerprint, "pg_stat_monitor_reset()"):
			continue
		case strings.Contains(b.Common.Example, "pg_stat_monitor_reset()"):
			continue
		case strings.Contains(b.Common.Example, "pgstatstatements"):
			continue
		default:
			res = append(res, b)
		}
	}
	return res
}

func TestVersion(t *testing.T) {
	pgsmVersion, err := ver.NewVersion("1.0.0-beta-2")
	require.NoError(t, err)
	require.True(t, pgsmVersion.LessThan(v10))
}

func TestPGStatMonitorSchema(t *testing.T) {
	t.Skip("Skip it until the sandbox supports pg_stat_monitor by default. The current PostgreSQL image is the official, not the one from PerconaLab")
	sqlDB := tests.OpenTestPostgreSQL(t)
	defer sqlDB.Close() //nolint:errcheck
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	engineVersion := tests.PostgreSQLVersion(t, sqlDB)
	if !supportedVersion(engineVersion) || !extensionExists(db) {
		t.Skip()
	}

	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_stat_monitor SCHEMA public")
	assert.NoError(t, err)

	defer func() {
		_, err = db.Exec("DROP EXTENSION pg_stat_monitor")
		assert.NoError(t, err)
	}()

	vPG, err := getPGVersion(db.Querier)
	assert.NoError(t, err)

	vPGSM, _, err := getPGMonitorVersion(db.Querier)
	assert.NoError(t, err)

	_, view := newPgStatMonitorStructs(vPGSM, vPG)
	structs, err := db.SelectAllFrom(view, "")
	require.NoError(t, err)
	tests.LogTable(t, structs)

	const selectAllCountries = "SELECT /* AllCountries:PGStatMonitor controller='test' */ * FROM country"
	const selectAllCountriesLong = "SELECT /* AllCountriesTruncated:PGStatMonitor controller='test' */ * FROM country WHERE capital IN " +
		"($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, " +
		"$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, " +
		"$41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59, $60, " +
		"$61, $62, $63, $64, $65, $66, $67, $68, $69, $70, $71, $72, $73, $74, $75, $76, $77, $78, $79, $80, " +
		"$81, $82, $83, $84, $85, $86, $87, $88, $89, $90, $91, $92, $93, $94, $95, $96, $97, $98, $99, $100, " +
		"$101, $102, $103, $104, $105, $106, $107, $108, $109, $110, $111, $112, $113, $114, $115, $116, $117, $118, $119, $120, " +
		"$121, $122, $123, $124, $125, $126, $127, $128, $129, $130, $131, $132, $133, $134, $135, $136, $137, $138, $139, $140, " +
		"$141, $142, $143, $144, $145, $146, $147, $148, $149, $150, $151, $152, $153, $154, $155, $156, $157, $158, $159, $160, " +
		"$161, $162, $163, $164, $165, $166, $167, $168, $169, $170, $171, $172, $173, $174, $175, $176, $177, $178, $179, $180, " +
		"$181, $182, $183, $184, $185, $186, $187, $188, $189, $190, $191, $192, $193, $194, $195, $196, $197, $198, $199, $200, " +
		"$201, $202, $203, $204, $205, $206, $207, $208, $209, $210, $211, $212, $213, $214, $215, $216, $217, $218, $219, $220, " +
		"$221, $222, $223, $224, $225, $226, $227, $228, $229, $230, $231, $232, $233, $234, $235, $236, $237, $238, $239, $240, " +
		"$241, $242, $243, $244, $245, $246, $247, $248, $249, $250, $251, $252, $253, $254, $255, $256, $257, $258, $259, $260, " +
		"$261, $262, $263, $264, $265, $266, $267, $268, $269, $270, $271, $272, $273, $274, $275, $276, $277, $278, $279, $280, " +
		"$281, $282, $283, $284, $285, $286, $287, $288, $289, $290, $291, $292, $293, $294, $295, $296, $297, $298, $299, $300, " +
		"$301, $302, $303, $304, $305, $306, $307, $308, $309, $310, $311, $312, $313, $314, $315, $316, $317, $318, $319, $320, " +
		"$321, $322, $323, $324, $325, $326, $327, $328, $329, $330, $331, $332, $333, $334, $335, $336, $337, $338, $339, $340, " +
		"$341, $342, $343, $3 ..."

	var digests map[string]string
	switch engineVersion {
	case "11":
		digests = map[string]string{
			selectAllCountries:     "8055E3FCBD5A55B1",
			selectAllCountriesLong: "FD567C4A01A1FC5C",
		}
	case "12":
		digests = map[string]string{
			selectAllCountries:     "4D9388B06139847E",
			selectAllCountriesLong: "1BD274D6C4EFDEAF",
		}

	case "13":
		digests = map[string]string{
			selectAllCountries:     "6DB095DC2F1199D9",
			selectAllCountriesLong: "7FE44699CA927E67",
		}

	case "14", "15":
		digests = map[string]string{
			selectAllCountries:     "1172613B04E8CEC0",
			selectAllCountriesLong: "EB821DA6C4030A4F",
		}
	default:
		t.Log("Unhandled version, assuming dummy digests.")
		digests = map[string]string{
			selectAllCountries:     "TODO-selectAllCountries",
			selectAllCountriesLong: "TODO-selectAllCountriesLong",
		}
	}

	var selectCMDType, insertCMDType string
	var mPlansCallsCnt, mPlansTimeCnt float32
	pgsmVersion, _, err := getPGMonitorVersion(db.Querier)
	assert.NoError(t, err)
	switch pgsmVersion {
	case pgStatMonitorVersion06:
	case pgStatMonitorVersion08:
	case pgStatMonitorVersion09:
		selectCMDType = commandTypeSelect
		insertCMDType = commandTypeInsert
	case pgStatMonitorVersion10PG12,
		pgStatMonitorVersion11PG12,
		pgStatMonitorVersion20PG12:
		selectCMDType = commandTypeSelect
		insertCMDType = commandTypeInsert
	case pgStatMonitorVersion10PG13, pgStatMonitorVersion10PG14,
		pgStatMonitorVersion11PG13, pgStatMonitorVersion11PG14,
		pgStatMonitorVersion20PG13, pgStatMonitorVersion20PG14, pgStatMonitorVersion20PG15:
		selectCMDType = commandTypeSelect
		insertCMDType = commandTypeInsert
		mPlansCallsCnt = 1
		mPlansTimeCnt = 1
	}

	t.Run("AllCountries", func(t *testing.T) {
		m := setup(t, db, false, false)

		_, err := db.Exec(selectAllCountries)
		require.NoError(t, err)

		settings, err := m.getSettings()
		require.NoError(t, err)
		normalizedQuery, err := settings.getNormalizedQueryValue()
		require.NoError(t, err)

		buckets, err := m.getNewBuckets(context.Background(), 60, normalizedQuery)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual := buckets[0]
		actual.Common.Username = strings.ReplaceAll(actual.Common.Username, `"`, "")
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.Equal(t, float32(5), actual.Postgresql.MSharedBlksHitSum+actual.Postgresql.MSharedBlksReadSum)
		assert.InDelta(t, 1.5, actual.Postgresql.MSharedBlksHitCnt+actual.Postgresql.MSharedBlksReadCnt, 0.5)
		example := ""

		if !normalizedQuery && !m.disableQueryExamples {
			example = actual.Common.Example
		}

		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCountries,
				Example:             example,
				ExampleType:         agentpb.ExampleType_RANDOM,
				Database:            "pmm-agent",
				Tables:              []string{"public.country"},
				Comments:            map[string]string{"controller": "test"},
				Username:            "pmm-agent",
				ClientHost:          actual.Common.ClientHost,
				AgentId:             "agent_id",
				PeriodStartUnixSecs: actual.Common.PeriodStartUnixSecs,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MSharedBlkReadTimeCnt: actual.Postgresql.MSharedBlkReadTimeCnt,
				MSharedBlkReadTimeSum: actual.Postgresql.MSharedBlkReadTimeSum,
				MLocalBlkReadTimeCnt:  actual.Postgresql.MLocalBlkReadTimeCnt,
				MLocalBlkReadTimeSum:  actual.Postgresql.MLocalBlkReadTimeSum,
				MSharedBlksReadCnt:    actual.Postgresql.MSharedBlksReadCnt,
				MSharedBlksReadSum:    actual.Postgresql.MSharedBlksReadSum,
				MSharedBlksHitCnt:     actual.Postgresql.MSharedBlksHitCnt,
				MSharedBlksHitSum:     actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:              1,
				MRowsSum:              239,
				MCpuUserTimeCnt:       actual.Postgresql.MCpuUserTimeCnt,
				MCpuUserTimeSum:       actual.Postgresql.MCpuUserTimeSum,
				MCpuSysTimeCnt:        actual.Postgresql.MCpuSysTimeCnt,
				MCpuSysTimeSum:        actual.Postgresql.MCpuSysTimeSum,
				CmdType:               selectCMDType,
				HistogramItems:        actual.Postgresql.HistogramItems,
				MPlansCallsSum:        actual.Postgresql.MPlansCallsSum,
				MPlansCallsCnt:        mPlansCallsCnt,
				MPlanTimeCnt:          mPlansTimeCnt,
				MPlanTimeSum:          actual.Postgresql.MPlanTimeSum,
				MPlanTimeMin:          actual.Postgresql.MPlanTimeMin,
				MPlanTimeMax:          actual.Postgresql.MPlanTimeMax,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MSharedBlkReadTimeSum, actual.Common.MQueryTimeSum)
		assert.Regexp(t, `\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}`, actual.Common.ClientHost)

		_, err = db.Exec(selectAllCountries)
		require.NoError(t, err)

		buckets, err = m.getNewBuckets(context.Background(), 60, normalizedQuery)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual = buckets[0]
		actual.Common.Username = strings.ReplaceAll(actual.Common.Username, `"`, "")
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		expected = &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCountries,
				Example:             example,
				ExampleType:         agentpb.ExampleType_RANDOM,
				Database:            "pmm-agent",
				Tables:              []string{"public.country"},
				Comments:            map[string]string{"controller": "test"},
				Username:            "pmm-agent",
				ClientHost:          actual.Common.ClientHost,
				AgentId:             "agent_id",
				PeriodStartUnixSecs: actual.Common.PeriodStartUnixSecs,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MSharedBlksHitCnt:     1,
				MSharedBlksHitSum:     5,
				MRowsCnt:              1,
				MRowsSum:              239,
				MSharedBlkReadTimeCnt: actual.Postgresql.MSharedBlkReadTimeCnt,
				MSharedBlkReadTimeSum: actual.Postgresql.MSharedBlkReadTimeSum,
				MLocalBlkReadTimeCnt:  actual.Postgresql.MLocalBlkReadTimeCnt,
				MLocalBlkReadTimeSum:  actual.Postgresql.MLocalBlkReadTimeSum,
				MCpuUserTimeCnt:       actual.Postgresql.MCpuUserTimeCnt,
				MCpuUserTimeSum:       actual.Postgresql.MCpuUserTimeSum,
				MCpuSysTimeCnt:        actual.Postgresql.MCpuSysTimeCnt,
				MCpuSysTimeSum:        actual.Postgresql.MCpuSysTimeSum,
				CmdType:               selectCMDType,
				HistogramItems:        actual.Postgresql.HistogramItems,
				MPlansCallsSum:        actual.Postgresql.MPlansCallsSum,
				MPlansCallsCnt:        mPlansCallsCnt,
				MPlanTimeCnt:          mPlansTimeCnt,
				MPlanTimeSum:          actual.Postgresql.MPlanTimeSum,
				MPlanTimeMin:          actual.Postgresql.MPlanTimeMin,
				MPlanTimeMax:          actual.Postgresql.MPlanTimeMax,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MSharedBlkReadTimeSum, actual.Common.MQueryTimeSum)
	})

	t.Run("AllCountriesTruncated", func(t *testing.T) {
		m := setup(t, db, false, false)

		const n = 500
		placeholders := db.Placeholders(1, n)
		args := make([]interface{}, n)
		for i := 0; i < n; i++ {
			args[i] = i
		}
		q := fmt.Sprintf("SELECT /* AllCountriesTruncated:PGStatMonitor controller='test' */ * FROM country WHERE capital IN (%s)", strings.Join(placeholders, ", "))
		_, err := db.Exec(q, args...)
		require.NoError(t, err)

		settings, err := m.getSettings()
		require.NoError(t, err)
		normalizedQuery, err := settings.getNormalizedQueryValue()
		require.NoError(t, err)

		buckets, err := m.getNewBuckets(context.Background(), 60, normalizedQuery)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual := buckets[0]
		actual.Common.Username = strings.ReplaceAll(actual.Common.Username, `"`, "")
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 5, actual.Postgresql.MSharedBlksHitSum+actual.Postgresql.MSharedBlksReadSum, 3)
		assert.InDelta(t, 1.5, actual.Postgresql.MSharedBlksHitCnt+actual.Postgresql.MSharedBlksReadCnt, 0.5)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCountriesLong,
				Example:             actual.Common.Example,
				ExampleType:         agentpb.ExampleType_RANDOM,
				Database:            "pmm-agent",
				Tables:              []string{"public.country"},
				Comments:            map[string]string{"controller": "test"},
				Username:            "pmm-agent",
				ClientHost:          actual.Common.ClientHost,
				AgentId:             "agent_id",
				PeriodStartUnixSecs: actual.Common.PeriodStartUnixSecs,
				PeriodLengthSecs:    60,
				IsTruncated:         true,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MSharedBlkReadTimeCnt: actual.Postgresql.MSharedBlkReadTimeCnt,
				MSharedBlkReadTimeSum: actual.Postgresql.MSharedBlkReadTimeSum,
				MLocalBlkReadTimeCnt:  actual.Postgresql.MLocalBlkReadTimeCnt,
				MLocalBlkReadTimeSum:  actual.Postgresql.MLocalBlkReadTimeSum,
				MSharedBlksReadCnt:    actual.Postgresql.MSharedBlksReadCnt,
				MSharedBlksReadSum:    actual.Postgresql.MSharedBlksReadSum,
				MSharedBlksHitCnt:     actual.Postgresql.MSharedBlksHitCnt,
				MSharedBlksHitSum:     actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:              1,
				MRowsSum:              30,
				MCpuUserTimeCnt:       actual.Postgresql.MCpuUserTimeCnt,
				MCpuUserTimeSum:       actual.Postgresql.MCpuUserTimeSum,
				MCpuSysTimeCnt:        actual.Postgresql.MCpuSysTimeCnt,
				MCpuSysTimeSum:        actual.Postgresql.MCpuSysTimeSum,
				CmdType:               selectCMDType,
				HistogramItems:        actual.Postgresql.HistogramItems,
				MPlansCallsSum:        actual.Postgresql.MPlansCallsSum,
				MPlansCallsCnt:        mPlansCallsCnt,
				MPlanTimeCnt:          mPlansTimeCnt,
				MPlanTimeSum:          actual.Postgresql.MPlanTimeSum,
				MPlanTimeMin:          actual.Postgresql.MPlanTimeMin,
				MPlanTimeMax:          actual.Postgresql.MPlanTimeMax,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MSharedBlkReadTimeSum, actual.Common.MQueryTimeSum)
		assert.Regexp(t, `\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}`, actual.Common.ClientHost)

		_, err = db.Exec(q, args...)
		require.NoError(t, err)

		buckets, err = m.getNewBuckets(context.Background(), 60, normalizedQuery)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual = buckets[0]
		actual.Common.Username = strings.ReplaceAll(actual.Common.Username, `"`, "")
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.Postgresql.MSharedBlkReadTimeCnt, 1)
		assert.InDelta(t, 5, actual.Postgresql.MSharedBlksHitSum, 2)
		expected = &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCountriesLong,
				Example:             actual.Common.Example,
				ExampleType:         agentpb.ExampleType_RANDOM,
				Database:            "pmm-agent",
				Tables:              []string{"public.country"},
				Comments:            map[string]string{"controller": "test"},
				Username:            "pmm-agent",
				ClientHost:          actual.Common.ClientHost,
				AgentId:             "agent_id",
				PeriodStartUnixSecs: actual.Common.PeriodStartUnixSecs,
				PeriodLengthSecs:    60,
				IsTruncated:         true,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MSharedBlkReadTimeCnt: actual.Postgresql.MSharedBlkReadTimeCnt,
				MSharedBlkReadTimeSum: actual.Postgresql.MSharedBlkReadTimeSum,
				MLocalBlkReadTimeCnt:  actual.Postgresql.MLocalBlkReadTimeCnt,
				MLocalBlkReadTimeSum:  actual.Postgresql.MLocalBlkReadTimeSum,
				MSharedBlksHitCnt:     1,
				MSharedBlksHitSum:     actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:              1,
				MRowsSum:              30,
				MCpuUserTimeCnt:       actual.Postgresql.MCpuUserTimeCnt,
				MCpuUserTimeSum:       actual.Postgresql.MCpuUserTimeSum,
				MCpuSysTimeCnt:        actual.Postgresql.MCpuSysTimeCnt,
				MCpuSysTimeSum:        actual.Postgresql.MCpuSysTimeSum,
				CmdType:               selectCMDType,
				HistogramItems:        actual.Postgresql.HistogramItems,
				MPlansCallsSum:        actual.Postgresql.MPlansCallsSum,
				MPlansCallsCnt:        mPlansCallsCnt,
				MPlanTimeCnt:          mPlansTimeCnt,
				MPlanTimeSum:          actual.Postgresql.MPlanTimeSum,
				MPlanTimeMin:          actual.Postgresql.MPlanTimeMin,
				MPlanTimeMax:          actual.Postgresql.MPlanTimeMax,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MSharedBlkReadTimeSum, actual.Common.MQueryTimeSum)
	})

	t.Run("CheckMBlkReadTime", func(t *testing.T) {
		r := rand.New(rand.NewSource(time.Now().Unix())) //nolint:gosec
		tableName := fmt.Sprintf("customer%d", r.Int())
		_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			customer_id integer NOT NULL,
			first_name character varying(45) NOT NULL,
			last_name character varying(45) NOT NULL,
			active boolean
		)`, tableName))
		require.NoError(t, err)
		defer func() {
			_, err := db.Exec(fmt.Sprintf(`DROP TABLE %s`, tableName))
			require.NoError(t, err)
		}()
		m := setup(t, db, false, false)

		var waitGroup sync.WaitGroup
		n := 1000
		for i := 0; i < n; i++ {
			id := i
			query := fmt.Sprintf(`INSERT /* CheckMBlkReadTime controller='test' */ INTO %s (customer_id, first_name, last_name, active) VALUES (%d, 'John', 'Dow', TRUE)`, tableName, id)
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				_, err := db.Exec(query)
				require.NoError(t, err)
			}()
		}
		waitGroup.Wait()

		settings, err := m.getSettings()
		require.NoError(t, err)
		normalizedQuery, err := settings.getNormalizedQueryValue()
		require.NoError(t, err)

		var buckets []*agentpb.MetricsBucket
		for i := 0; i < 100; i++ {
			buckets, err = m.getNewBuckets(context.Background(), 60, normalizedQuery)
			require.NoError(t, err)
			buckets = filter(buckets)
			t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
			if len(buckets) != 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.Len(t, buckets, 1)

		actual := buckets[0]
		actual.Common.Username = strings.ReplaceAll(actual.Common.Username, `"`, "")
		assert.NotZero(t, actual.Postgresql.MSharedBlkReadTimeSum)
		expectedFingerprint := fmt.Sprintf("INSERT /* CheckMBlkReadTime controller='test' */ INTO %s (customer_id, first_name, last_name, active) VALUES ($1, $2, $3, $4)", tableName)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Queryid:             actual.Common.Queryid,
				Fingerprint:         expectedFingerprint,
				Example:             actual.Common.Example,
				ExampleType:         agentpb.ExampleType_RANDOM,
				Comments:            map[string]string{"controller": "test"},
				Database:            "pmm-agent",
				Username:            "pmm-agent",
				ClientHost:          actual.Common.ClientHost,
				AgentId:             "agent_id",
				PeriodStartUnixSecs: actual.Common.PeriodStartUnixSecs,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT,
				NumQueries:          float32(n),
				MQueryTimeCnt:       float32(n),
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
				// FIXME: Why tables is empty here? this will error.
				Tables: []string{fmt.Sprintf("public.%s", tableName)},
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MSharedBlkReadTimeCnt: float32(n),
				MSharedBlkReadTimeSum: actual.Postgresql.MSharedBlkReadTimeSum,
				MLocalBlkReadTimeCnt:  actual.Postgresql.MLocalBlkReadTimeCnt,
				MLocalBlkReadTimeSum:  actual.Postgresql.MLocalBlkReadTimeSum,
				MSharedBlksReadCnt:    actual.Postgresql.MSharedBlksReadCnt,
				MSharedBlksReadSum:    actual.Postgresql.MSharedBlksReadSum,
				MSharedBlksWrittenCnt: actual.Postgresql.MSharedBlksWrittenCnt,
				MSharedBlksWrittenSum: actual.Postgresql.MSharedBlksWrittenSum,
				MSharedBlksDirtiedCnt: actual.Postgresql.MSharedBlksDirtiedCnt,
				MSharedBlksDirtiedSum: actual.Postgresql.MSharedBlksDirtiedSum,
				MSharedBlksHitCnt:     actual.Postgresql.MSharedBlksHitCnt,
				MSharedBlksHitSum:     actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:              float32(n),
				MRowsSum:              float32(n),
				MCpuUserTimeCnt:       actual.Postgresql.MCpuUserTimeCnt,
				MCpuUserTimeSum:       actual.Postgresql.MCpuUserTimeSum,
				MCpuSysTimeCnt:        actual.Postgresql.MCpuSysTimeCnt,
				MCpuSysTimeSum:        actual.Postgresql.MCpuSysTimeSum,
				CmdType:               insertCMDType,
				HistogramItems:        actual.Postgresql.HistogramItems,
				MPlansCallsSum:        actual.Postgresql.MPlansCallsSum,
				MPlansCallsCnt:        actual.Postgresql.MPlansCallsCnt,
				MPlanTimeCnt:          actual.Postgresql.MPlanTimeCnt,
				MPlanTimeSum:          actual.Postgresql.MPlanTimeSum,
				MPlanTimeMin:          actual.Postgresql.MPlanTimeMin,
				MPlanTimeMax:          actual.Postgresql.MPlanTimeMax,
				MWalBytesCnt:          actual.Postgresql.MWalBytesCnt,
				MWalBytesSum:          actual.Postgresql.MWalBytesSum,
				MWalRecordsSum:        actual.Postgresql.MWalRecordsSum,
				MWalRecordsCnt:        actual.Postgresql.MWalRecordsCnt,
			},
		}
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MSharedBlkReadTimeSum, actual.Common.MQueryTimeSum)
		assert.Regexp(t, `\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}`, actual.Common.ClientHost)
	})
}
