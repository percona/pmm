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

// Package client contains business logic of working with pmm-managed.
package client

import "time"

type connectionSet struct {
	events       []connectionEvent
	windowPeriod time.Duration
}

type connectionEvent struct {
	t         time.Time
	connected bool
}

func NewConnectionSet(windowPeriod time.Duration) *connectionSet {
	return &connectionSet{
		windowPeriod: windowPeriod,
	}
}

func (c *connectionSet) Set(timestamp time.Time, connnected bool) {
	c.deleteOldEvents()

	newElem := connectionEvent{
		t:         timestamp,
		connected: connnected,
	}

	if len(c.events) != 0 {
		lastElem := c.events[len(c.events)-1]
		if lastElem.connected != connnected {
			c.events = append(c.events, newElem)
		}
	} else {
		c.events = append(c.events, newElem)
	}
}

func (c *connectionSet) deleteOldEvents() {
	for i, e := range c.events {
		if e.t.Before(time.Now().Add(-1 * c.windowPeriod)) {
			c.events = append(c.events[:i], c.events[i+1:]...)
		}
	}
}

func (c *connectionSet) GetAll() []connectionEvent {
	return c.events
}
