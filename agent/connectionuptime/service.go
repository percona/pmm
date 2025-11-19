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

// Package connectionuptime contains functionality for connection uptime calculation.
package connectionuptime

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const periodForRunningDeletingOldEvents = time.Minute

// Service calculates connection uptime between agent and server
// based on the connection events.
type Service struct {
	mx           sync.RWMutex
	events       []connectionEvent
	windowPeriod time.Duration
	l            *logrus.Entry
}

type connectionEvent struct {
	Timestamp time.Time
	Connected bool
}

// NewService creates new instance of Service.
func NewService(windowPeriod time.Duration) *Service {
	return &Service{
		windowPeriod: windowPeriod,
		l:            logrus.WithField("component", "connection-uptime-service"),
	}
}

// SetWindowPeriod updates window period.
func (c *Service) SetWindowPeriod(windowPeriod time.Duration) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.windowPeriod = windowPeriod
}

// RegisterConnectionStatus adds connection event.
func (c *Service) RegisterConnectionStatus(timestamp time.Time, connected bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.addEvent(timestamp, connected)
}

func (c *Service) addEvent(timestamp time.Time, connected bool) {
	if len(c.events) == 0 || c.events[len(c.events)-1].Connected != connected {
		c.events = append(c.events, connectionEvent{
			Timestamp: timestamp,
			Connected: connected,
		})
	}
}

func (c *Service) deleteOldEvents(toTime time.Time) {
	c.mx.Lock()
	defer c.mx.Unlock()

	if len(c.events) == 0 {
		return
	}

	// Move first elements which are already expired to the start of the slice
	// in order to not lose information about previous state of connection.
	// The latest expired element in the slice will be the first one to calculate
	// uptime correctly during set up window time
	boundaryTimestamp := toTime.Add(-c.windowPeriod)
	for i := len(c.events) - 1; i >= 0; i-- {
		if c.events[i].Timestamp.Before(boundaryTimestamp) {
			c.removeFirstElementsUntilIndex(i)

			return
		}
	}
}

func (c *Service) removeFirstElementsUntilIndex(i int) {
	c.events = append(c.events[:0], c.events[i:]...)
}

// RunCleanupGoroutine starts goroutine which removes already expired connection events.
// Expired event means that it was created more than `windowPeriod` time ago.
func (c *Service) RunCleanupGoroutine(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(periodForRunningDeletingOldEvents)
		for {
			select {
			case <-ticker.C:
				c.l.Debug("Called delete old events")
				c.deleteOldEvents(time.Now())
			case <-ctx.Done():
				c.l.Debug("Done")
				return
			}
		}
	}()
}

// GetConnectedUpTimeUntil calculates the connection uptime between agent and server
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
// Method will calculate connected time as interval between connected and disconnected events
//
// Here is example how it works.
// When we have such set of events in connection set `f1 s1 f2`
// where f1 - first event of failed connection
//
//	s1 - first event of successful connection
//	f2 - second event of failed connection
//
// method will return result using next formula `time_between(s1, f2)/time_between(f1, now)*100`
// where time_between(s1, f2) - connection up time
//
//	time_between(f1, now) - total time between first event (f1) and current moment
func (c *Service) GetConnectedUpTimeUntil(toTime time.Time) float32 {
	c.mx.RLock()
	defer c.mx.RUnlock()

	c.l.Debug("Calculate connection uptime")
	if len(c.events) == 0 {
		return 0
	}

	if len(c.events) == 1 {
		if c.events[0].Connected {
			return 100
		}
		return 0
	}

	var connectedTimeMs int64
	expiredTime := toTime.Add(-c.windowPeriod)
	for i, event := range c.events {
		if event.Connected {
			from := maxTime(expiredTime, event.Timestamp)

			var to time.Time
			if i+1 >= len(c.events) {
				to = toTime
			} else {
				to = maxTime(expiredTime, c.events[i+1].Timestamp)
			}

			// don't consider both events which are already expired
			if from.After(to) || from.Equal(to) {
				continue
			}

			connectedTimeMs += to.UnixMilli() - from.UnixMilli()
		}
	}

	startTime := maxTime(expiredTime, c.events[0].Timestamp)
	totalTime := toTime.Sub(startTime).Milliseconds()

	return float32(connectedTimeMs) / float32(totalTime) * 100
}

func maxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
