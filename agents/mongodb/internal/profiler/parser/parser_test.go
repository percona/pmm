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

package parser

import (
	"reflect"
	"testing"
	"time"

	pm "github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/aggregator"
	"github.com/percona/pmm-agent/agents/mongodb/internal/report"
)

func TestNew(t *testing.T) {
	docsChan := make(chan pm.SystemProfile)
	a := aggregator.New(time.Now(), "test-id")

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
			want: New(docsChan, a),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.docsChan, tt.args.aggregator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New(%v, %v) = %v, want %v", tt.args.docsChan, tt.args.aggregator, got, tt.want)
			}
		})
	}
}

func TestParserStartStop(t *testing.T) {
	var err error
	docsChan := make(chan pm.SystemProfile)
	a := aggregator.New(time.Now(), "test-id")

	parser1 := New(docsChan, a)
	err = parser1.Start()
	require.NoError(t, err)

	// running multiple Start() should be idempotent
	err = parser1.Start()
	require.NoError(t, err)

	// running multiple Stop() should be idempotent
	parser1.Stop()
	parser1.Stop()
}

func TestParserrunning(t *testing.T) {
	docsChan := make(chan pm.SystemProfile)
	a := aggregator.New(time.Now(), "test-id")
	reportChan := a.Start()
	defer a.Stop()
	d := aggregator.DefaultInterval

	parser1 := New(docsChan, a)
	err := parser1.Start()
	require.NoError(t, err)

	now := time.Now().UTC()
	timeStart := now.Truncate(d).Add(d)
	timeEnd := timeStart.Add(d)

	select {
	case docsChan <- pm.SystemProfile{
		Ts: timeStart,
		Query: pm.BsonD{
			{"find", "test"},
		},
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
			StartTs: timeStart,
			EndTs:   timeEnd,
		}

		assert.Equal(t, expected.StartTs, actual.StartTs)
		assert.Equal(t, expected.EndTs, actual.EndTs)
		assert.Len(t, actual.Buckets, 1)
		assert.EqualValues(t, actual.Buckets[0].Common.NumQueries, 1)

	case <-time.After(d + 5*time.Second):
		t.Error("test timeout")
	}

	parser1.Stop()
}
