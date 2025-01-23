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

package aggregator

import (
	"context"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/percona/pmm/agent/agents/mongodb/internal/profiler/collector"
	"github.com/percona/pmm/agent/agents/mongodb/internal/report"
	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestAggregator(t *testing.T) {
	// we need at least one test per package to correctly calculate coverage
	t.Run("Add", func(t *testing.T) {
		t.Run("error if aggregator is not running", func(t *testing.T) {
			a := New(time.Now(), "test-agent", logrus.WithField("component", "test"), truncate.GetMongoDBDefaultMaxQueryLength())
			err := a.Add(context.TODO(), collector.ExtendedSystemProfile{})
			assert.EqualError(t, err, "aggregator is not running")
		})
	})

	t.Run("createResult", func(t *testing.T) {
		agentID := "test-agent"
		startPeriod := time.Now()
		aggregator := New(startPeriod, agentID, logrus.WithField("component", "test"), truncate.GetMongoDBDefaultMaxQueryLength())
		aggregator.Start()
		defer aggregator.Stop()
		ctx := context.TODO()
		err := aggregator.Add(ctx, collector.ExtendedSystemProfile{
			SystemProfile: proto.SystemProfile{
				NscannedObjects: 2,
				Nreturned:       3,
				Ns:              "collection.people",
				Op:              "insert",
			},
		})
		require.NoError(t, err)

		result := aggregator.createResult(ctx)

		require.Equal(t, 1, len(result.Buckets))
		assert.Equal(t, report.Result{
			Buckets: []*agentv1.MetricsBucket{
				{
					Common: &agentv1.MetricsBucket_Common{
						Queryid:             result.Buckets[0].Common.Queryid,
						Fingerprint:         "INSERT people",
						Database:            "collection",
						Tables:              []string{"people"},
						AgentId:             agentID,
						AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT,
						PeriodStartUnixSecs: uint32(startPeriod.Truncate(DefaultInterval).Unix()),
						PeriodLengthSecs:    60,
						Example:             `{"ns":"collection.people","op":"insert"}`,
						ExampleType:         agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
						NumQueries:          1,
						MQueryTimeCnt:       1,
					},
					Mongodb: &agentv1.MetricsBucket_MongoDB{
						MDocsReturnedCnt:   1,
						MDocsReturnedSum:   3,
						MDocsReturnedMin:   3,
						MDocsReturnedMax:   3,
						MDocsReturnedP99:   3,
						MResponseLengthCnt: 1,
						MDocsScannedCnt:    1,
						MDocsScannedSum:    2,
						MDocsScannedMin:    2,
						MDocsScannedMax:    2,
						MDocsScannedP99:    2,
					},
				},
			},
		}, *result)
	})

	t.Run("createResultInvalidUTF8", func(t *testing.T) {
		agentID := "test-agent"
		startPeriod := time.Now()
		aggregator := New(startPeriod, agentID, logrus.WithField("component", "test"), truncate.GetMongoDBDefaultMaxQueryLength())
		aggregator.Start()
		defer aggregator.Stop()
		ctx := context.TODO()
		err := aggregator.Add(ctx, collector.ExtendedSystemProfile{
			SystemProfile: proto.SystemProfile{
				NscannedObjects: 2,
				Nreturned:       3,
				Ns:              "collection.people",
				Op:              "query",
				Command: bson.D{
					primitive.E{Key: "find", Value: "people"},
					primitive.E{
						Key: "filter",
						Value: bson.D{
							primitive.E{Key: "name_\xff", Value: "value_\xff"},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		result := aggregator.createResult(ctx)

		require.Equal(t, 1, len(result.Buckets))
		assert.True(t, utf8.ValidString(result.Buckets[0].Common.Example))
		assert.Equal(t, report.Result{
			Buckets: []*agentv1.MetricsBucket{
				{
					Common: &agentv1.MetricsBucket_Common{
						Queryid:             result.Buckets[0].Common.Queryid,
						Fingerprint:         "FIND people name_\ufffd",
						Database:            "collection",
						Tables:              []string{"people"},
						AgentId:             agentID,
						AgentType:           inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT,
						PeriodStartUnixSecs: uint32(startPeriod.Truncate(DefaultInterval).Unix()),
						PeriodLengthSecs:    60,
						Example:             "{\"ns\":\"collection.people\",\"op\":\"query\",\"command\":{\"find\":\"people\",\"filter\":{\"name_\\ufffd\":\"value_\\ufffd\"}}}",
						ExampleType:         agentv1.ExampleType_EXAMPLE_TYPE_RANDOM,
						NumQueries:          1,
						MQueryTimeCnt:       1,
					},
					Mongodb: &agentv1.MetricsBucket_MongoDB{
						MDocsReturnedCnt:   1,
						MDocsReturnedSum:   3,
						MDocsReturnedMin:   3,
						MDocsReturnedMax:   3,
						MDocsReturnedP99:   3,
						MResponseLengthCnt: 1,
						MDocsScannedCnt:    1,
						MDocsScannedSum:    2,
						MDocsScannedMin:    2,
						MDocsScannedMax:    2,
						MDocsScannedP99:    2,
					},
				},
			},
		}, *result)
	})
}
