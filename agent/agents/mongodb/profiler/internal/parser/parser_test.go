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

package parser

import (
	"context"
	"reflect"
	"testing"
	"time"

	pm "github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/report"
	"github.com/percona/pmm/agent/utils/truncate"
)

func TestNew(t *testing.T) {
	docsChan := make(chan pm.SystemProfile)
	a := aggregator.New(time.Now(), "test-id", logrus.WithField("component", "aggregator"), truncate.GetMongoDBDefaultMaxQueryLength())

	type args struct {
		docsChan   <-chan pm.SystemProfile
		aggregator *aggregator.Aggregator
	}
	tests := []struct {
		name string
		args args
		want *Parser
	}{
		{
			name: "TestNew",
			args: args{
				docsChan:   docsChan,
				aggregator: a,
			},
			want: New(docsChan, a, logrus.WithField("component", "test-parser")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.docsChan, tt.args.aggregator, logrus.WithField("component", "test-parser")); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New(%v, %v) = %v, want %v", tt.args.docsChan, tt.args.aggregator, got, tt.want)
			}
		})
	}
}

func TestParserStartStop(t *testing.T) {
	var err error
	docsChan := make(chan pm.SystemProfile)
	a := aggregator.New(time.Now(), "test-id", logrus.WithField("component", "aggregator"), truncate.GetMongoDBDefaultMaxQueryLength())

	ctx := context.TODO()
	parser1 := New(docsChan, a, logrus.WithField("component", "test-parser"))
	err = parser1.Start(ctx)
	require.NoError(t, err)

	// running multiple Start() should be idempotent
	err = parser1.Start(ctx)
	require.NoError(t, err)

	// running multiple Stop() should be idempotent
	parser1.Stop()
	parser1.Stop()
}

func TestParserRunning(t *testing.T) {
	oldInterval := aggregator.DefaultInterval
	aggregator.DefaultInterval = 10 * time.Second
	defer func() { aggregator.DefaultInterval = oldInterval }()
	docsChan := make(chan pm.SystemProfile)
	a := aggregator.New(time.Now(), "test-id", logrus.WithField("component", "aggregator"), truncate.GetMongoDBDefaultMaxQueryLength())
	reportChan := a.Start()
	defer a.Stop()
	d := aggregator.DefaultInterval

	parser1 := New(docsChan, a, logrus.WithField("component", "test-parser"))
	err := parser1.Start(context.TODO())
	require.NoError(t, err)
	defer parser1.Stop()

	now := time.Now().UTC()
	timeStart := now.Truncate(d).Add(d)
	timeEnd := timeStart.Add(d)

	select {
	case docsChan <- pm.SystemProfile{
		Ns: "test.test",
		Ts: timeStart,
		Query: bson.D{
			{"find", "test"},
		},
		Op:             "query",
		ResponseLength: 100,
		DocsExamined:   200,
		Nreturned:      300,
		Millis:         4000,
	}:
	case <-time.After(5 * time.Second):
		t.Error("test timeout")
	}

	sp := pm.SystemProfile{
		Ts: timeEnd.Add(1 * time.Second),
	}
	select {
	case docsChan <- sp:
	case <-time.After(5 * time.Second):
		t.Error("test timeout")
	}

	select {
	case actual := <-reportChan:
		expected := report.Report{
			StartTS: timeStart,
			EndTS:   timeEnd,
		}

		assert.Equal(t, expected.StartTS, actual.StartTS)
		assert.Equal(t, expected.EndTS, actual.EndTS)
		assert.Len(t, actual.Buckets, 1)
		assert.InEpsilon(t, 1, actual.Buckets[0].Common.NumQueries, 0.001)

	case <-time.After(d + 5*time.Second):
		t.Error("test timeout")
	}
}
