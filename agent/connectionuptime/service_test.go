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

package connectionuptime

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetElemsToConnectionSet(t *testing.T) {
	const (
		dayPeriod = 24 * time.Hour
	)

	now := time.Now()

	tests := []struct {
		name           string
		windowPeriod   time.Duration
		args           map[time.Time]bool
		expectedEvents []ConnectionEvent
	}{
		{
			name:         "should return one event",
			windowPeriod: dayPeriod,
			args: map[time.Time]bool{
				now: true,
			},
			expectedEvents: []ConnectionEvent{
				{
					Timestamp: now,
					Connected: true,
				},
			},
		},
		{
			name:         "should return only one event when we have sequence of same events",
			windowPeriod: dayPeriod,
			args: map[time.Time]bool{
				now:                      true,
				now.Add(time.Minute):     true,
				now.Add(1 * time.Minute): true,
				now.Add(2 * time.Minute): true,
			},
			expectedEvents: []ConnectionEvent{
				{
					Timestamp: now,
					Connected: true,
				},
			},
		},
		{
			name:         "should return set of events",
			windowPeriod: dayPeriod,
			args: map[time.Time]bool{
				now:                      true,
				now.Add(1 * time.Minute): true,
				now.Add(2 * time.Minute): false,
				now.Add(3 * time.Minute): false,
				now.Add(4 * time.Minute): false,
				now.Add(5 * time.Minute): false,
				now.Add(6 * time.Minute): true,
				now.Add(7 * time.Minute): true,
			},
			expectedEvents: []ConnectionEvent{
				{
					Timestamp: now,
					Connected: true,
				},
				{
					Timestamp: now.Add(2 * time.Minute),
					Connected: false,
				},
				{
					Timestamp: now.Add(6 * time.Minute),
					Connected: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewService(tt.windowPeriod)

			var sortedTime []time.Time
			for k := range tt.args {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				set.AddConnectionEvent(t, tt.args[t])
			}

			assert.Equal(t, tt.expectedEvents, set.GetAll())
		})
	}
}

func TestConnectionUpTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name             string
		setOfConnections map[time.Time]bool
		expectedUpTime   float32
	}{
		{
			name: "should be 100%",
			setOfConnections: map[time.Time]bool{
				now: true,
			},
			expectedUpTime: 100,
		},
		{
			name: "should be 0%",
			setOfConnections: map[time.Time]bool{
				now: false,
			},
			expectedUpTime: 0,
		},
		{
			name: "should be 50% when half of the time there is no connection between server and server",
			setOfConnections: map[time.Time]bool{
				now.Add(-10 * time.Second): false,
				now.Add(-5 * time.Second):  true,
			},
			expectedUpTime: 50,
		},
		{
			name: "should be 10% when only 6 seconds was uptime from 1 minute",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Minute): false,
				now.Add(-6 * time.Second): true,
			},
			expectedUpTime: 10,
		},
		{
			name: "should be 90% when only 54 seconds was uptime from 1 minute",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Minute): true,
				now.Add(-6 * time.Second): false,
			},
			expectedUpTime: 90,
		},
		{
			name: "should be 50% when only 30 seconds was uptime from 1 minute",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Minute):  true,
				now.Add(-50 * time.Second): false,
				now.Add(-40 * time.Second): false,
				now.Add(-20 * time.Second): true,
			},
			expectedUpTime: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewService(time.Hour)
			var sortedTime []time.Time
			for k := range tt.setOfConnections {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				cs.AddConnectionEvent(t, tt.setOfConnections[t])
			}

			assert.EqualValues(t, tt.expectedUpTime, cs.GetConnectedUpTimeSince(now))
		})
	}
}

func TestConnectionSet_DeleteOldEvents(t *testing.T) {
	now := time.Now()

	type fields struct {
		events       map[time.Time]bool
		windowPeriod time.Duration
	}
	tests := []struct {
		name     string
		fields   fields
		expected []ConnectionEvent
	}{
		{
			name: "should return remove expired element with connected=true and created instead of it a new one",
			fields: fields{
				events: map[time.Time]bool{
					now.Add(-61 * time.Minute): true,
					now.Add(-1 * time.Minute):  false,
				},
				windowPeriod: time.Hour,
			},
			expected: []ConnectionEvent{
				{
					Timestamp: now.Add(-1 * time.Hour).Add(time.Second),
					Connected: true,
				},
				{
					Timestamp: now.Add(-1 * time.Minute),
					Connected: false,
				},
			},
		},
		{
			name: "should return remove expired element with connected=false and created instead of it a new one",
			fields: fields{
				events: map[time.Time]bool{
					now.Add(-61 * time.Minute): false,
					now.Add(-1 * time.Minute):  true,
				},
				windowPeriod: time.Hour,
			},
			expected: []ConnectionEvent{
				{
					Timestamp: now.Add(-1 * time.Hour).Add(time.Second),
					Connected: false,
				},
				{
					Timestamp: now.Add(-1 * time.Minute),
					Connected: true,
				},
			},
		},
		{
			name: "should remove expired element with connected=false and replace it with the next one",
			fields: fields{
				events: map[time.Time]bool{
					now.Add(-121 * time.Minute): false,
					now.Add(-60 * time.Minute):  true,
				},
				windowPeriod: time.Hour,
			},
			expected: []ConnectionEvent{
				{
					Timestamp: now.Add(-60 * time.Minute).Add(time.Second),
					Connected: true,
				},
			},
		},
		{
			name: "should update single event which is expired",
			fields: fields{
				events: map[time.Time]bool{
					now.Add(-121 * time.Minute): false,
				},
				windowPeriod: time.Hour,
			},
			expected: []ConnectionEvent{
				{
					Timestamp: now.Add(-60 * time.Minute).Add(time.Second),
					Connected: false,
				},
			},
		},
		{
			name: "should update single event which is expired",
			fields: fields{
				events: map[time.Time]bool{
					now.Add(-121 * time.Minute): false,
					now.Add(-120 * time.Minute): true,
					now.Add(-119 * time.Minute): false,
					now.Add(-118 * time.Minute): true,
					now.Add(-117 * time.Minute): false,
					now.Add(-116 * time.Minute): true,
				},
				windowPeriod: time.Hour,
			},
			expected: []ConnectionEvent{
				{
					Timestamp: now.Add(-60 * time.Minute).Add(time.Second),
					Connected: true,
				},
			},
		},
		{
			name: "should update single event which is expired",
			fields: fields{
				events: map[time.Time]bool{
					now.Add(-121 * time.Minute): false,
					now.Add(-120 * time.Minute): true,
					now.Add(-119 * time.Minute): false,
					now.Add(-60 * time.Minute):  true,
					now.Add(-59 * time.Minute):  false,
					now.Add(-58 * time.Minute):  true,
				},
				windowPeriod: time.Hour,
			},
			expected: []ConnectionEvent{
				{
					Timestamp: now.Add(-60 * time.Minute).Add(time.Second),
					Connected: true,
				},
				{
					Timestamp: now.Add(-59 * time.Minute),
					Connected: false,
				},
				{
					Timestamp: now.Add(-58 * time.Minute),
					Connected: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewService(tt.fields.windowPeriod)

			var sortedTime []time.Time
			for k := range tt.fields.events {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				cs.AddConnectionEvent(t, tt.fields.events[t])
			}

			cs.deleteOldEvents()

			gotEvents := cs.GetAll()
			assert.True(t, len(gotEvents) == len(tt.expected), "length of got slice of events is not correct")
			for i, e := range gotEvents {
				assert.Equal(t, tt.expected[i].Timestamp.Unix(), e.Timestamp.Unix(), fmt.Sprintf("element with index: %d", i))
				assert.Equal(t, tt.expected[i].Connected, e.Connected, fmt.Sprintf("element with index: %d", i))
			}
		})
	}
}
