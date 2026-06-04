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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOffsetSchedule(t *testing.T) {
	t.Run("assigns unique offsets while slots are available", func(t *testing.T) {
		s := newOffsetSchedule()
		seen := make(map[time.Duration]bool)

		for i := range 60 {
			offset, release := s.assign(fmt.Sprintf("agent-%d", i), time.Minute)
			defer release()

			require.False(t, seen[offset])
			seen[offset] = true
		}
	})

	t.Run("reuses offsets only after all slots are occupied", func(t *testing.T) {
		s := newOffsetSchedule()
		counts := make(map[time.Duration]int)

		for i := range 61 {
			offset, release := s.assign(fmt.Sprintf("agent-%d", i), time.Minute)
			defer release()

			counts[offset]++
		}

		assert.Len(t, counts, 60)
		for _, count := range counts {
			assert.LessOrEqual(t, count, 2)
		}
	})

	t.Run("distributes offsets evenly", func(t *testing.T) {
		for _, agentsCount := range []int{60, 120, 180, 181} {
			s := newOffsetSchedule()
			counts := make(map[time.Duration]int)

			for i := range agentsCount {
				offset, release := s.assign(fmt.Sprintf("agent-%d", i), time.Minute)
				defer release()

				counts[offset]++
			}

			require.Len(t, counts, 60)
			minCount := agentsCount
			maxCount := 0
			for _, count := range counts {
				minCount = min(minCount, count)
				maxCount = max(maxCount, count)
			}
			assert.LessOrEqual(t, maxCount-minCount, 1)
		}
	})

	t.Run("releases offsets", func(t *testing.T) {
		s := newOffsetSchedule()
		offset, release := s.assign("agent-1", time.Minute)

		release()
		assert.Empty(t, s.offset)
		assert.Empty(t, s.counts)

		newOffset, newRelease := s.assign("agent-2", time.Second)
		defer newRelease()

		assert.Equal(t, offset%time.Second, newOffset)
	})
}
