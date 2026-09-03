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

package agents

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

var minuteSchedule = newOffsetSchedule()

type offsetSchedule struct {
	m      sync.Mutex
	offset map[string]int
	counts map[int]int
}

func newOffsetSchedule() *offsetSchedule {
	return &offsetSchedule{
		offset: make(map[string]int),
		counts: make(map[int]int),
	}
}

// RandomMinuteOffset assigns a random delay within a minute.
// Offsets are unique while there are no more than 60 active agents.
func RandomMinuteOffset(agentID string) (time.Duration, func()) {
	return minuteSchedule.assign(agentID, time.Minute)
}

// DelayUntilOffset returns the time left before the offset slot in the current interval.
func DelayUntilOffset(now time.Time, interval, offset time.Duration) time.Duration {
	if interval <= 0 || offset <= 0 {
		return 0
	}

	offset %= interval
	periodStart := now.Truncate(interval)
	target := periodStart.Add(offset)
	if !target.After(now) {
		return 0
	}

	return target.Sub(now)
}

func (s *offsetSchedule) assign(agentID string, interval time.Duration) (time.Duration, func()) {
	s.m.Lock()
	defer s.m.Unlock()

	if offset, ok := s.offset[agentID]; ok {
		return time.Duration(offset) * time.Second, func() {}
	}

	slots := int(interval / time.Second)
	if slots <= 0 {
		return 0, func() {}
	}

	offset := s.nextOffset(slots)
	s.offset[agentID] = offset
	s.counts[offset]++

	var once sync.Once
	release := func() {
		once.Do(func() {
			s.release(agentID)
		})
	}

	return time.Duration(offset) * time.Second, release
}

func (s *offsetSchedule) nextOffset(slots int) int {
	minCount := int(^uint(0) >> 1)
	candidates := make([]int, 0, slots)
	for i := range slots {
		count := s.counts[i]
		switch {
		case count < minCount:
			minCount = count
			candidates = candidates[:0]
			candidates = append(candidates, i)
		case count == minCount:
			candidates = append(candidates, i)
		}
	}

	return candidates[randomInt(len(candidates))]
}

func (s *offsetSchedule) release(agentID string) {
	s.m.Lock()
	defer s.m.Unlock()

	offset, ok := s.offset[agentID]
	if !ok {
		return
	}

	delete(s.offset, agentID)
	s.counts[offset]--
	if s.counts[offset] <= 0 {
		delete(s.counts, offset)
	}
}

func randomInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return int(time.Now().UnixNano() % int64(n))
	}

	return int(v.Int64())
}
