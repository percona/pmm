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

package slowlog

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/percona/go-mysql/event"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/utils/tests"
)

func getDataFromFile(t *testing.T, filePath string, data interface{}) {
	jsonData, err := ioutil.ReadFile(filePath) //nolint:gosec
	require.NoError(t, err)
	err = json.Unmarshal(jsonData, &data)
	require.NoError(t, err)
}

func TestSlowLogMakeBuckets(t *testing.T) {
	t.Parallel()

	const agentID = "/agent_id/73ee2f92-d5aa-45f0-8b09-6d3df605fd44"
	periodStart := time.Unix(1557137220, 0)

	parsingResult := event.Result{}
	getDataFromFile(t, "slowlog_fixture.json", &parsingResult)

	actualBuckets := makeBuckets(agentID, parsingResult, periodStart, 60, false)

	expectedBuckets := []*agentpb.MetricsBucket{}
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
	db := tests.OpenTestMySQL(t)
	defer db.Close() //nolint:errcheck
	_, vendor := tests.MySQLVersion(t, db)

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
		if vendor == tests.PerconaMySQL {
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
		for range s.Changes() {
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
		if vendor == tests.PerconaMySQL {
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
		for range s.Changes() {
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
		if vendor == tests.PerconaMySQL {
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
		for range s.Changes() {
		}
	})
}
