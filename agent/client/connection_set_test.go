package client

import (
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
		expectedEvents []connectionEvent
	}{
		{
			name:         "should return one event",
			windowPeriod: dayPeriod,
			args: map[time.Time]bool{
				now: true,
			},
			expectedEvents: []connectionEvent{
				{
					t:         now,
					connected: true,
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
			expectedEvents: []connectionEvent{
				{
					t:         now,
					connected: true,
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
			expectedEvents: []connectionEvent{
				{
					t:         now,
					connected: true,
				},
				{
					t:         now.Add(2 * time.Minute),
					connected: false,
				},
				{
					t:         now.Add(6 * time.Minute),
					connected: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewConnectionSet(tt.windowPeriod)

			var sortedTime []time.Time
			for k := range tt.args {
				sortedTime = append(sortedTime, k)
			}

			sort.Slice(sortedTime, func(i, j int) bool {
				return sortedTime[i].Before(sortedTime[j])
			})

			for _, t := range sortedTime {
				set.Set(t, tt.args[t])
			}

			assert.Equal(t, tt.expectedEvents, set.GetAll())
		})
	}
}

func TestConnectionSetExpirationElements(t *testing.T) {
	const secondPeriod = time.Second

	t.Run("should not return element if it is expired", func(t *testing.T) {
		now := time.Now()

		set := NewConnectionSet(secondPeriod)

		set.Set(now, true)
		time.Sleep(2 * time.Second)
		// after expiration of window time first element should be removed when we set
		// new time
		set.Set(now.Add(1*time.Minute), true)

		expectedEvents := []connectionEvent{
			{
				t:         now.Add(1 * time.Minute),
				connected: true,
			},
		}

		assert.Equal(t, expectedEvents, set.GetAll())
	})
}
