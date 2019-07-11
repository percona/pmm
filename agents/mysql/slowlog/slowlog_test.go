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

package slowlog

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/percona/go-mysql/event"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/agents"
	"github.com/percona/pmm-agent/utils/tests"
)

func assertBucketsEqual(t *testing.T, expected, actual *qanpb.MetricsBucket) bool {
	t.Helper()
	return assert.Equal(t, proto.MarshalTextString(expected), proto.MarshalTextString(actual))
}

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

	actualBuckets := makeBuckets(agentID, parsingResult, periodStart, 60)

	expectedBuckets := []*qanpb.MetricsBucket{}
	getDataFromFile(t, "slowlog_expected.json", &expectedBuckets)

	countActualBuckets := len(actualBuckets)
	countExpectedBuckets := 0
	for _, actualBucket := range actualBuckets {
		for _, expectedBucket := range expectedBuckets {
			if actualBucket.Queryid == expectedBucket.Queryid {
				assertBucketsEqual(t, expectedBucket, actualBucket)
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

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		wd, err := os.Getwd()
		require.NoError(t, err)
		params := &Params{
			DSN:               tests.GetTestMySQLDSN(t),
			SlowLogFilePrefix: filepath.Join(wd, "..", "..", "..", "testdata"),
		}
		s, err := New(params, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expectedInfo := &slowLogInfo{
			path: "/slowlogs/slow.log",
		}
		if vendor == tests.PerconaMySQL {
			expectedInfo.outlierTime = 10
		}

		actualInfo, err := s.getSlowLogInfo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedInfo, actualInfo)

		_, err = os.Stat(filepath.Join(params.SlowLogFilePrefix, "/slowlogs/slow.log"))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		go s.Run(ctx)

		expected := []agents.Change{
			{Status: inventorypb.AgentStatus_STARTING},
			{Status: inventorypb.AgentStatus_RUNNING},
			{Status: inventorypb.AgentStatus_WAITING},
		}
		actual := []agents.Change{
			<-s.Changes(),
			<-s.Changes(),
			<-s.Changes(),
		}
		assert.Equal(t, expected, actual)

		cancel()
		for range s.Changes() {
		}
	})

	t.Run("NoFile", func(t *testing.T) {
		t.Parallel()

		params := &Params{
			DSN: tests.GetTestMySQLDSN(t),
		}
		s, err := New(params, logrus.WithField("test", t.Name()))
		require.NoError(t, err)

		expectedInfo := &slowLogInfo{
			path: "/slowlogs/slow.log",
		}
		if vendor == tests.PerconaMySQL {
			expectedInfo.outlierTime = 10
		}

		actualInfo, err := s.getSlowLogInfo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedInfo, actualInfo)

		_, err = os.Stat(filepath.Join(params.SlowLogFilePrefix, "/slowlogs/slow.log"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		go s.Run(ctx)

		expected := []agents.Change{
			{Status: inventorypb.AgentStatus_STARTING},
			{Status: inventorypb.AgentStatus_WAITING},
			{Status: inventorypb.AgentStatus_STARTING},
		}
		actual := []agents.Change{
			<-s.Changes(),
			<-s.Changes(),
			<-s.Changes(),
		}
		assert.Equal(t, expected, actual)

		cancel()
		for range s.Changes() {
		}
	})
}
