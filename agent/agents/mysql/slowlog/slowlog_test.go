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

package slowlog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/percona/go-mysql/event"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/truncate"
	"github.com/percona/pmm/agent/utils/version"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

func getDataFromFile(t *testing.T, filePath string, data interface{}) {
	t.Helper()

	jsonData, err := os.ReadFile(filePath) //nolint:gosec
	require.NoError(t, err)
	err = json.Unmarshal(jsonData, &data)
	require.NoError(t, err)
}

func TestSlowLogMakeBucketsInvalidUTF8(t *testing.T) {
	const agentID = "/agent_id/73ee2f92-d5aa-45f0-8b09-6d3df605fd44"
	periodStart := time.Unix(1557137220, 0)

	parsingResult := event.Result{
		Class: map[string]*event.Class{
			"example": {
				Metrics:     &event.Metrics{},
				Fingerprint: "SELECT /* controller='test' */ * FROM contacts t0 WHERE t0.person_id = '߿�\xff\\ud83d\xdd'",
				Example: &event.Example{
					Query: "SELECT /* controller='test' */ * FROM contacts t0 WHERE t0.person_id = '߿�\xff\\ud83d\xdd'",
				},
			},
		},
	}

	actualBuckets := makeBuckets(agentID, parsingResult, periodStart, 60, false, false, truncate.GetDefaultMaxQueryLength(), logrus.NewEntry(logrus.New()))
	expectedBuckets := []*agentpb.MetricsBucket{
		{
			Common: &agentpb.MetricsBucket_Common{
				Fingerprint:         "select * from contacts t0 where t0.person_id = ?",
				ExplainFingerprint:  "select * from contacts t0 where t0.person_id = :1",
				PlaceholdersCount:   1,
				Comments:            map[string]string{"controller": "test"},
				AgentId:             agentID,
				AgentType:           inventorypb.AgentType_QAN_MYSQL_SLOWLOG_AGENT,
				PeriodStartUnixSecs: 1557137220,
				PeriodLengthSecs:    60,
				Example:             "SELECT /* controller='test' */ * FROM contacts t0 WHERE t0.person_id = '߿�\ufffd\\ud83d\ufffd'",
				ExampleType:         agentpb.ExampleType_RANDOM,
			},
			Mysql: &agentpb.MetricsBucket_MySQL{},
		},
	}

	require.Equal(t, 1, len(actualBuckets))
	assert.True(t, utf8.ValidString(actualBuckets[0].Common.Example))
	tests.AssertBucketsEqual(t, expectedBuckets[0], actualBuckets[0])
}

func TestSlowLogMakeBuckets(t *testing.T) {
	t.Parallel()

	const agentID = "/agent_id/73ee2f92-d5aa-45f0-8b09-6d3df605fd44"
	periodStart := time.Unix(1557137220, 0)

	parsingResult := event.Result{}
	getDataFromFile(t, "slowlog_fixture.json", &parsingResult)

	actualBuckets := makeBuckets(agentID, parsingResult, periodStart, 60, false, false, truncate.GetDefaultMaxQueryLength(), logrus.NewEntry(logrus.New()))

	var expectedBuckets []*agentpb.MetricsBucket
	getDataFromFile(t, "slowlog_expected.json", &expectedBuckets)

	countActualBuckets := len(actualBuckets)
	countExpectedBuckets := 0
	for _, actualBucket := range actualBuckets {
		for _, expectedBucket := range expectedBuckets {
			if actualBucket.Common.Queryid == expectedBucket.Common.Queryid {
				tests.AssertBucketsEqual(t, expectedBucket, actualBucket)
				countExpectedBuckets++
			}
		}
	}
	assert.Equal(t, countExpectedBuckets, countActualBuckets)
}

func TestSlowLog(t *testing.T) {
	t.Parallel()
	sqlDB := tests.OpenTestMySQL(t)
	t.Cleanup(func() { sqlDB.Close() }) //nolint:errcheck

	q := reform.NewDB(sqlDB, mysql.Dialect, reform.NewPrintfLogger(t.Logf)).WithTag(queryTag)
	ctx := context.Background()
	_, vendor, _ := version.GetMySQLVersion(ctx, q)

	testdata, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata"))
	require.NoError(t, err)

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		params := &Params{
			DSN:               tests.GetTestMySQLDSN(t),
			SlowLogFilePrefix: testdata,
		}
		s, err := New(params, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expectedInfo := &slowLogInfo{
			path: "/mysql/slowlogs/slow.log",
		}
		if vendor == version.PerconaVendor {
			expectedInfo.outlierTime = 10
		}

		actualInfo, err := s.getSlowLogInfo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedInfo, actualInfo)

		_, err = os.Stat(filepath.Join(params.SlowLogFilePrefix, "/mysql/slowlogs/slow.log"))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		go s.Run(ctx)

		// collect first 3 status changes, skip QAN data
		var actual []inventorypb.AgentStatus
		for c := range s.Changes() {
			if c.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
				actual = append(actual, c.Status)
				if len(actual) == 3 {
					break
				}
			}
		}

		expected := []inventorypb.AgentStatus{
			inventorypb.AgentStatus_STARTING,
			inventorypb.AgentStatus_RUNNING,
			inventorypb.AgentStatus_WAITING,
		}
		assert.Equal(t, expected, actual)

		cancel()
		for range s.Changes() { //nolint:revive
		}
	})

	t.Run("NoFile", func(t *testing.T) {
		t.Parallel()

		params := &Params{
			DSN:               tests.GetTestMySQLDSN(t),
			SlowLogFilePrefix: "nonexistent",
		}
		s, err := New(params, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expectedInfo := &slowLogInfo{
			path: "/mysql/slowlogs/slow.log",
		}
		if vendor == version.PerconaVendor {
			expectedInfo.outlierTime = 10
		}

		actualInfo, err := s.getSlowLogInfo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedInfo, actualInfo)

		_, err = os.Stat(filepath.Join(params.SlowLogFilePrefix, "/mysql/slowlogs/slow.log"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		go s.Run(ctx)

		// collect first 3 status changes, skip QAN data
		var actual []inventorypb.AgentStatus
		for c := range s.Changes() {
			if c.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
				actual = append(actual, c.Status)
				if len(actual) == 3 {
					break
				}
			}
		}

		expected := []inventorypb.AgentStatus{
			inventorypb.AgentStatus_STARTING,
			inventorypb.AgentStatus_WAITING,
			inventorypb.AgentStatus_STARTING,
		}
		assert.Equal(t, expected, actual)

		cancel()
		for range s.Changes() { //nolint:revive
		}
	})

	t.Run("NormalWithRotation", func(t *testing.T) {
		params := &Params{
			DSN:                tests.GetTestMySQLDSN(t),
			MaxSlowlogFileSize: 1,
			SlowLogFilePrefix:  testdata,
		}
		s, err := New(params, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expectedInfo := &slowLogInfo{
			path: "/mysql/slowlogs/slow.log",
		}
		if vendor == version.PerconaVendor {
			expectedInfo.outlierTime = 10
		}

		actualInfo, err := s.getSlowLogInfo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedInfo, actualInfo)

		_, err = os.Stat(filepath.Join(params.SlowLogFilePrefix, "/mysql/slowlogs/slow.log"))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		go s.Run(ctx)

		// collect first 3 status changes, skip QAN data
		var actual []inventorypb.AgentStatus
		for c := range s.Changes() {
			if c.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
				actual = append(actual, c.Status)
				if len(actual) == 3 {
					break
				}
			}
		}

		expected := []inventorypb.AgentStatus{
			inventorypb.AgentStatus_STARTING,
			inventorypb.AgentStatus_RUNNING,
			inventorypb.AgentStatus_WAITING,
		}
		assert.Equal(t, expected, actual)

		cancel()
		for range s.Changes() { //nolint:revive
		}
	})
}
