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

package pgstatstatements

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

func setup(t *testing.T, db *reform.DB) *PGStatStatementsQAN {
	t.Helper()

	selectQuery := fmt.Sprintf("SELECT /* %s */ ", queryTag)

	_, err := db.Exec(selectQuery + "pg_stat_statements_reset()")
	require.NoError(t, err)

	p, err := newPgStatStatementsQAN(db.WithTag(queryTag), nil, "agent_id", logrus.WithField("test", t.Name()))
	require.NoError(t, err)

	return p
}

// filter removes buckets for queries that are not expected by tests.
func filter(mb []*agentpb.MetricsBucket) []*agentpb.MetricsBucket {
	res := make([]*agentpb.MetricsBucket, 0, len(mb))
	for _, b := range mb {
		switch {
		case strings.Contains(b.Common.Fingerprint, "/* pmm-agent:pgstatstatements */"):
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

	defer func() {
		_, err := db.Exec("DROP EXTENSION pg_stat_statements")
		assert.NoError(t, err)
	}()

	structs, err := db.SelectAllFrom(pgStatDatabaseView, "")
	require.NoError(t, err)
	rows, err := rowsByVersion(db.Querier, "")
	require.NoError(t, err)

	defer func() {
		e := rows.Close()
		if err == nil {
			err = e
		}
	}()

	for {
		str := pgStatStatementsView.NewStruct()
		if err = db.Querier.NextRow(str, rows); err != nil {
			break
		}

		structs = append(structs, str)
	}
	if errors.Is(err, reform.ErrNoRows) {
		err = nil
	}
	require.NoError(t, err)
	tests.LogTable(t, structs)

	const selectAllCities = "SELECT /* AllCities:pgstatstatements */ * FROM city"
	const selectAllCitiesLong = "SELECT /* AllCitiesTruncated:pgstatstatements */ * FROM city WHERE id IN " +
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
		"$341, $342, $343, $344, $345, $346, $3 ..."

	engineVersion := tests.PostgreSQLVersion(t, sqlDB)
	var digests map[string]string // digest_text/fingerprint to digest/query_id
	switch engineVersion {
	case "9.4":
		digests = map[string]string{
			selectAllCities:     "3239586867",
			selectAllCitiesLong: "2745128652",
		}
	case "9.5", "9.6":
		digests = map[string]string{
			selectAllCities:     "3994135135",
			selectAllCitiesLong: "2677760328",
		}
	case "10":
		digests = map[string]string{
			selectAllCities:     "2229807896",
			selectAllCitiesLong: "3454929487",
		}
	case "11":
		digests = map[string]string{
			selectAllCities:     "-4056421706168012289",
			selectAllCitiesLong: "2233640464962569536",
		}
	case "12":
		digests = map[string]string{
			selectAllCities:     "5627444073676588515",
			selectAllCitiesLong: "-1605123213815583414",
		}
	case "13":
		digests = map[string]string{
			selectAllCities:     "-32455482996301954",
			selectAllCitiesLong: "-4813789842463369261",
		}
	case "14":
		digests = map[string]string{
			selectAllCities:     "5991662752016701281",
			selectAllCitiesLong: "-3564720362103294944",
		}
	default:
		t.Log("Unhandled version, assuming dummy digests.")
		digests = map[string]string{
			selectAllCities:     "TODO-selectAllCities",
			selectAllCitiesLong: "TODO-selectAllCitiesLong",
		}
	}

	// Need to detect vendor because result for mSharedBlksReadSum are different for different images for postgres.
	mSharedBlksHitSum := float32(33)
	if strings.Contains(os.Getenv("POSTGRES_IMAGE"), "perconalab") {
		mSharedBlksHitSum = 32
	}

	t.Run("AllCities", func(t *testing.T) {
		m := setup(t, db)

		_, err := db.Exec(selectAllCities)
		require.NoError(t, err)

		buckets, err := m.getNewBuckets(context.Background(), time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual := buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.Equal(t, mSharedBlksHitSum, actual.Postgresql.MSharedBlksHitSum+actual.Postgresql.MSharedBlksReadSum)
		assert.InDelta(t, 1.5, actual.Postgresql.MSharedBlksHitCnt+actual.Postgresql.MSharedBlksReadCnt, 0.5)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCities,
				Database:            "pmm-agent",
				Tables:              []string{"city"},
				Username:            "pmm-agent",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MBlkReadTimeCnt:    actual.Postgresql.MBlkReadTimeCnt,
				MBlkReadTimeSum:    actual.Postgresql.MBlkReadTimeSum,
				MSharedBlksReadCnt: actual.Postgresql.MSharedBlksReadCnt,
				MSharedBlksReadSum: actual.Postgresql.MSharedBlksReadSum,
				MSharedBlksHitCnt:  actual.Postgresql.MSharedBlksHitCnt,
				MSharedBlksHitSum:  actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:           1,
				MRowsSum:           4079,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MBlkReadTimeSum, actual.Common.MQueryTimeSum)

		_, err = db.Exec(selectAllCities)
		require.NoError(t, err)

		buckets, err = m.getNewBuckets(context.Background(), time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual = buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		expected = &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCities,
				Database:            "pmm-agent",
				Tables:              []string{"city"},
				Username:            "pmm-agent",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MSharedBlksHitCnt: 1,
				MSharedBlksHitSum: mSharedBlksHitSum,
				MRowsCnt:          1,
				MRowsSum:          4079,
				MBlkReadTimeCnt:   actual.Postgresql.MBlkReadTimeCnt,
				MBlkReadTimeSum:   actual.Postgresql.MBlkReadTimeSum,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MBlkReadTimeSum, actual.Common.MQueryTimeSum)
	})

	t.Run("AllCitiesTruncated", func(t *testing.T) {
		m := setup(t, db)

		const n = 500
		placeholders := db.Placeholders(1, n)
		args := make([]interface{}, n)
		for i := 0; i < n; i++ {
			args[i] = i
		}
		q := fmt.Sprintf("SELECT /* AllCitiesTruncated:pgstatstatements */ * FROM city WHERE id IN (%s)", strings.Join(placeholders, ", "))
		_, err := db.Exec(q, args...)
		require.NoError(t, err)

		buckets, err := m.getNewBuckets(context.Background(), time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual := buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 1010, actual.Postgresql.MSharedBlksHitSum+actual.Postgresql.MSharedBlksReadSum, 3)
		assert.InDelta(t, 1.5, actual.Postgresql.MSharedBlksHitCnt+actual.Postgresql.MSharedBlksReadCnt, 0.5)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCitiesLong,
				Database:            "pmm-agent",
				Tables:              []string{},
				Username:            "pmm-agent",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				IsTruncated:         true,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MBlkReadTimeCnt:    actual.Postgresql.MBlkReadTimeCnt,
				MBlkReadTimeSum:    actual.Postgresql.MBlkReadTimeSum,
				MSharedBlksReadCnt: actual.Postgresql.MSharedBlksReadCnt,
				MSharedBlksReadSum: actual.Postgresql.MSharedBlksReadSum,
				MSharedBlksHitCnt:  actual.Postgresql.MSharedBlksHitCnt,
				MSharedBlksHitSum:  actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:           1,
				MRowsSum:           499,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MBlkReadTimeSum, actual.Common.MQueryTimeSum)

		_, err = db.Exec(q, args...)
		require.NoError(t, err)

		buckets, err = m.getNewBuckets(context.Background(), time.Date(2019, 4, 1, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		actual = buckets[0]
		assert.InDelta(t, 0, actual.Common.MQueryTimeSum, 0.09)
		assert.InDelta(t, 0, actual.Postgresql.MBlkReadTimeCnt, 1)
		assert.InDelta(t, 1007, actual.Postgresql.MSharedBlksHitSum, 2)
		expected = &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         selectAllCitiesLong,
				Database:            "pmm-agent",
				Tables:              []string{},
				Username:            "pmm-agent",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1554116340,
				PeriodLengthSecs:    60,
				IsTruncated:         true,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				NumQueries:          1,
				MQueryTimeCnt:       1,
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MBlkReadTimeCnt:   actual.Postgresql.MBlkReadTimeCnt,
				MBlkReadTimeSum:   actual.Postgresql.MBlkReadTimeSum,
				MSharedBlksHitCnt: 1,
				MSharedBlksHitSum: actual.Postgresql.MSharedBlksHitSum,
				MRowsCnt:          1,
				MRowsSum:          499,
			},
		}
		expected.Common.Queryid = digests[expected.Common.Fingerprint]
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MBlkReadTimeSum, actual.Common.MQueryTimeSum)
	})

	t.Run("CheckMBlkReadTime", func(t *testing.T) {
		r := rand.New(rand.NewSource(time.Now().Unix()))
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
		m := setup(t, db)

		var waitGroup sync.WaitGroup
		n := 1000
		for i := 0; i < n; i++ {
			id := i
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				_, err := db.Exec(fmt.Sprintf(`INSERT /* CheckMBlkReadTime */ INTO %s (customer_id, first_name, last_name, active) VALUES (%d, 'John', 'Dow', TRUE)`, tableName, id))
				require.NoError(t, err)
			}()
		}
		waitGroup.Wait()

		buckets, err := m.getNewBuckets(context.Background(), time.Date(2020, 5, 25, 10, 59, 0, 0, time.UTC), 60)
		require.NoError(t, err)
		buckets = filter(buckets)
		t.Logf("Actual:\n%s", tests.FormatBuckets(buckets))
		require.Len(t, buckets, 1)

		var fingerprint string
		switch engineVersion {
		case "9.4", "9.5", "9.6":
			fingerprint = fmt.Sprintf(`INSERT /* CheckMBlkReadTime */ INTO %s (customer_id, first_name, last_name, active) VALUES (?, ?, ?, ?)`, tableName)

		default:
			fingerprint = fmt.Sprintf(`INSERT /* CheckMBlkReadTime */ INTO %s (customer_id, first_name, last_name, active) VALUES ($1, $2, $3, $4)`, tableName)
		}
		actual := buckets[0]
		assert.NotZero(t, actual.Postgresql.MBlkReadTimeSum)
		expected := &agentpb.MetricsBucket{
			Common: &agentpb.MetricsBucket_Common{
				Queryid:             actual.Common.Queryid,
				Fingerprint:         fingerprint,
				Database:            "pmm-agent",
				Tables:              []string{tableName},
				Username:            "pmm-agent",
				AgentId:             "agent_id",
				PeriodStartUnixSecs: 1590404340,
				PeriodLengthSecs:    60,
				AgentType:           inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT,
				NumQueries:          float32(n),
				MQueryTimeCnt:       float32(n),
				MQueryTimeSum:       actual.Common.MQueryTimeSum,
			},
			Postgresql: &agentpb.MetricsBucket_PostgreSQL{
				MBlkReadTimeCnt:       float32(n),
				MBlkReadTimeSum:       actual.Postgresql.MBlkReadTimeSum,
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
			},
		}
		tests.AssertBucketsEqual(t, expected, actual)
		assert.LessOrEqual(t, actual.Postgresql.MBlkReadTimeSum, actual.Common.MQueryTimeSum)
	})
}
