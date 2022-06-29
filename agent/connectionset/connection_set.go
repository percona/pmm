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

package connectionset

import (
	"fmt"
	"sort"
	"time"
)

type ConnectionSet struct {
	events       []ConnectionEvent
	windowPeriod time.Duration
}

type ConnectionEvent struct {
	Timestamp time.Time
	Connected bool
}

func NewConnectionSet(windowPeriod time.Duration) *ConnectionSet {
	return &ConnectionSet{
		windowPeriod: windowPeriod,
	}
}

func (c *ConnectionSet) SetWindowPeriod(windowPeriod time.Duration) {
	c.windowPeriod = windowPeriod
}

func (c *ConnectionSet) Set(timestamp time.Time, connnected bool) {
	c.deleteOldEvents()

	newElem := ConnectionEvent{
		Timestamp: timestamp,
		Connected: connnected,
	}

	if len(c.events) != 0 {
		lastElem := c.events[len(c.events)-1]
		if lastElem.Connected != connnected {
			c.events = append(c.events, newElem)
		}
	} else {
		c.events = append(c.events, newElem)
	}
}

func (c *ConnectionSet) deleteOldEvents() {
	for i, e := range c.events {
		if e.Timestamp.Before(time.Now().Add(-1 * c.windowPeriod)) {
			c.events = append(c.events[:i], c.events[i+1:]...)
		}
	}
}

// GetConnectedUpTimeSince calculates the connection up time between agent and server
// based on the stored connection events.
//
// In the connection event set we store only when connection status was changed
// (was it connected or not) in the next format:
// {<timestamp, is_connected>, <timestamp, is_connected>, ...}
//
// For example:
// {<'2022-01-01 15:00:00', true>, <'2022-01-01 15:20:00', false>, <'2022-01-01 15:20:10', true>}
//
// GetConnectionUpTime returns the percentage of connection uptime during
// set period of time (by default it's 24 hours).
// Method will calculate connected time as interval between connected and disconneced events
//
// Here is example how it works.
// When we have such set of events in connection set `f1 s1 f2`
// where f1 - first event of failed connection
//       s1 - first event of successful connection
//       f2 - second event of failed connection
//
// method will return result using next formula `time_between(s1, f2)/time_between(f1, now)*100`
// where time_between(s1, f2) - connection up time
//       time_between(f1, now) - total time betweeen first event (f1) and current moment
func (c *ConnectionSet) GetConnectedUpTimeSince(sinceTime time.Time) float32 {
	if len(c.events) == 1 {
		if c.events[0].Connected {
			return 100
		} else {
			return 0
		}
	}
	// sort events by time
	sort.Slice(c.events, func(i, j int) bool {
		return c.events[i].Timestamp.Before(c.events[j].Timestamp)
	})

	fmt.Println(c.events)

	var connectedTimeMs int64
	for i, event := range c.events {
		if event.Connected {
			if i+1 >= len(c.events) {
				connectedTimeMs += sinceTime.Sub(event.Timestamp).Milliseconds()
			} else {
				connectedTimeMs += c.events[i+1].Timestamp.Sub(event.Timestamp).Milliseconds()
			}
		}
	}

	totalTime := sinceTime.Sub(c.events[0].Timestamp).Milliseconds()
	return float32(connectedTimeMs) / float32(totalTime) * 100
}

func (c *ConnectionSet) GetAll() []ConnectionEvent {
	return c.events
}
