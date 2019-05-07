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
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/percona/go-mysql/event"
	"github.com/percona/pmm/api/qanpb"
)

func assertBucketsEqual(t *testing.T, expected, actual *qanpb.MetricsBucket) bool {
	t.Helper()
	return assert.Equal(t, proto.MarshalTextString(expected), proto.MarshalTextString(actual))
}

func getDataFromFile(t *testing.T, filePath string, data interface{}) {
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Errorf("cannot read data from file:%s", err.Error())
	}
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		t.Errorf("cannot unmarshal json:%s", err.Error())
	}
}

func TestSlowLog(t *testing.T) {
	const agentID = "/agent_id/73ee2f92-d5aa-45f0-8b09-6d3df605fd44"
	ts := time.Unix(1557137220, 0)

	parsingResult := event.Result{}
	getDataFromFile(t, "slowlog_fixture.json", &parsingResult)

	actualBuckets := makeBuckets(agentID, parsingResult, ts)

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
