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

package connectionuptime

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectionUpTime(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name             string
		setOfConnections map[time.Time]bool
		expectedUpTime   float32
		windowPeriod     time.Duration
		toTime           time.Time
	}{
		{
			name: "should be 100%",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 100,
			windowPeriod:   time.Hour,
			toTime:         now,
		},
		{
			name: "should be 0%",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Second): false,
			},
			expectedUpTime: 0,
			windowPeriod:   time.Hour,
			toTime:         now,
		},
		{
			name: "should be 50% when half of the time there is no connection between server and server",
			setOfConnections: map[time.Time]bool{
				now.Add(-10 * time.Second): false,
				now.Add(-5 * time.Second):  true,
			},
			expectedUpTime: 50,
			windowPeriod:   time.Hour,
			toTime:         now,
		},
		{
			name: "should be 10% when only 6 seconds was uptime from 1 minute",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Minute): false,
				now.Add(-6 * time.Second): true,
			},
			expectedUpTime: 10,
			windowPeriod:   time.Hour,
			toTime:         now,
		},
		{
			name: "should be 90% when only 54 seconds was uptime from 1 minute",
			setOfConnections: map[time.Time]bool{
				now.Add(-1 * time.Minute): true,
				now.Add(-6 * time.Second): false,
			},
			expectedUpTime: 90,
			windowPeriod:   time.Hour,
			toTime:         now,
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
			windowPeriod:   time.Hour,
			toTime:         now,
		},
		{
			name: "should count uptime only during 1 minute and should be only 10% uptime",
			setOfConnections: map[time.Time]bool{
				now.Add(-2 * time.Minute):                false,
				now.Add(-1*time.Minute - 30*time.Second): true,
				now.Add(-1*time.Minute - 10*time.Second): false,
				now.Add(-6 * time.Second):                true,
			},
			expectedUpTime: 10,
			windowPeriod:   time.Minute,
			toTime:         now,
		},
		{
			name: "should return 100% uptime",
			setOfConnections: map[time.Time]bool{
				now.Add(-59 * time.Second): true,
				now.Add(-50 * time.Second): true,
				now.Add(-49 * time.Second): true,
				now.Add(-38 * time.Second): true,
				now.Add(-30 * time.Second): true,
				now.Add(-27 * time.Second): true,
				now.Add(-23 * time.Second): true,
				now.Add(-10 * time.Second): true,
			},
			expectedUpTime: 100,
			windowPeriod:   time.Minute,
			toTime:         now,
		},
		{
			name: "should return 100% uptime for 5 second period",
			setOfConnections: map[time.Time]bool{
				now.Add(-7 * time.Second): false,
				now.Add(-6 * time.Second): false,
				now.Add(-5 * time.Second): false,
				now.Add(-4 * time.Second): true,
				now.Add(-3 * time.Second): true,
				now.Add(-2 * time.Second): true,
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 80,
			windowPeriod:   5 * time.Second,
			toTime:         now,
		},
		{
			name:             "should return 0% uptime for 5 second period when there is no events",
			setOfConnections: nil,
			expectedUpTime:   0,
			windowPeriod:     5 * time.Second,
			toTime:           now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(tt.windowPeriod)

			var sortedTime []time.Time
			for k := range tt.setOfConnections {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				service.RegisterConnectionStatus(t, tt.setOfConnections[t])
			}

			service.deleteOldEvents(tt.toTime)
			got := service.GetConnectedUpTimeUntil(tt.toTime)
			assert.Truef(t, compareFloatWithTolerance(tt.expectedUpTime, got), "expected %f, got %f are not equal within tolerance", tt.expectedUpTime, got)
		})
	}
}

func TestConnectionUpTimeWithUpdatingConnectionUptime(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name             string
		setOfConnections map[time.Time]bool
		expectedUpTime   float32
		windowPeriod     time.Duration

		newExpectedUpTime float32
		newWindowPeriod   time.Duration
		toTime            time.Time
	}{
		{
			name: "should return 50% uptime when window period is 10s, and 100% uptime when window period is 5s",
			setOfConnections: map[time.Time]bool{
				now.Add(-8 * time.Second): false,
				now.Add(-7 * time.Second): false,
				now.Add(-6 * time.Second): false,
				now.Add(-5 * time.Second): false,
				now.Add(-4 * time.Second): true,
				now.Add(-3 * time.Second): true,
				now.Add(-2 * time.Second): true,
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 50,
			windowPeriod:   10 * time.Second,

			newExpectedUpTime: 80,
			newWindowPeriod:   5 * time.Second,
			toTime:            now,
		},
		{
			name: "should return 100% uptime when window period is 5s, and 100% uptime when window period is 10s",
			setOfConnections: map[time.Time]bool{
				now.Add(-4 * time.Second): true,
				now.Add(-3 * time.Second): true,
				now.Add(-2 * time.Second): true,
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 100,
			windowPeriod:   5 * time.Second,

			newExpectedUpTime: 100,
			newWindowPeriod:   5 * time.Second,
			toTime:            now,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(tt.windowPeriod)

			var sortedTime []time.Time
			for k := range tt.setOfConnections {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				service.RegisterConnectionStatus(t, tt.setOfConnections[t])
			}

			// delete expired events
			service.deleteOldEvents(tt.toTime)
			got := service.GetConnectedUpTimeUntil(tt.toTime)
			assert.Truef(t, compareFloatWithTolerance(tt.expectedUpTime, got), "expected %f, got %f are not equal within tolerance", tt.expectedUpTime, got)

			// updated window uptime
			service.SetWindowPeriod(tt.newWindowPeriod)

			// delete expired events
			service.deleteOldEvents(tt.toTime)
			got = service.GetConnectedUpTimeUntil(tt.toTime)
			assert.Truef(t, compareFloatWithTolerance(tt.newExpectedUpTime, got), "expected %f, got %f are not equal within tolerance", tt.newExpectedUpTime, got)
		})
	}
}

func compareFloatWithTolerance(a, b float32) bool {
	// here is big tolerance because sometimes we can't exactly know
	// the calculation uptime
	tolerance := 0.2
	diff := math.Abs(float64(a) - float64(b))

	return diff < tolerance
}

func TestCalculationConnectionUpTimeWhenCleanupMethodIsNotCalled(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name             string
		setOfConnections map[time.Time]bool
		expectedUpTime   float32
		windowPeriod     time.Duration
		toTime           time.Time
	}{
		{
			name: "should return 50% uptime when window period is 10s",
			setOfConnections: map[time.Time]bool{
				now.Add(-8 * time.Second): false,
				now.Add(-7 * time.Second): false,
				now.Add(-6 * time.Second): false,
				now.Add(-5 * time.Second): false,
				now.Add(-4 * time.Second): true,
				now.Add(-3 * time.Second): true,
				now.Add(-2 * time.Second): true,
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 50,
			windowPeriod:   10 * time.Second,
			toTime:         now,
		},
		{
			name: "should return 80% uptime when window period is 5s and when we have some events which are out of time window",
			setOfConnections: map[time.Time]bool{
				now.Add(-8 * time.Second): false,
				now.Add(-7 * time.Second): false,
				now.Add(-6 * time.Second): false,
				now.Add(-5 * time.Second): false,
				now.Add(-4 * time.Second): true,
				now.Add(-3 * time.Second): true,
				now.Add(-2 * time.Second): true,
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 80,
			windowPeriod:   5 * time.Second,
			toTime:         now,
		},
		{
			name: "should return 100% uptime when window period is 5s",
			setOfConnections: map[time.Time]bool{
				now.Add(-4 * time.Second): true,
				now.Add(-3 * time.Second): true,
				now.Add(-2 * time.Second): true,
				now.Add(-1 * time.Second): true,
			},
			expectedUpTime: 100,
			windowPeriod:   5 * time.Second,
			toTime:         now,
		},
		{
			name: "should return 80% uptime when window period is 5s and when we have some events which are out of time window",
			setOfConnections: map[time.Time]bool{
				now.Add(-10 * time.Second): false,
				now.Add(-4 * time.Second):  true,
				now.Add(-3 * time.Second):  true,
				now.Add(-2 * time.Second):  true,
				now.Add(-1 * time.Second):  true,
			},
			expectedUpTime: 80,
			windowPeriod:   5 * time.Second,
			toTime:         now,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(tt.windowPeriod)

			var sortedTime []time.Time
			for k := range tt.setOfConnections {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				service.RegisterConnectionStatus(t, tt.setOfConnections[t])
			}

			got := service.GetConnectedUpTimeUntil(tt.toTime)
			assert.Truef(t, compareFloatWithTolerance(tt.expectedUpTime, got), "expected %f, got %f are not equal within tolerance", tt.expectedUpTime, got)
		})
	}
}
