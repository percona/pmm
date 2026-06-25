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

package realtimeanalytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLockChains(t *testing.T) {
	t.Parallel()

	now := time.Now()
	sessions := map[int32]*sessionRow{
		100: {PID: 100, QueryStart: now.Add(-30 * time.Second)},
		200: {PID: 200, QueryStart: now.Add(-10 * time.Second)},
	}

	rows := []lockRow{{
		BlockedPID:   200,
		BlockerPID:   100,
		LockMode:     "ShareLock",
		RelationName: "orders",
		BlockerQuery: "UPDATE orders SET status = 1",
	}}

	chains := buildLockChains(rows, sessions)
	require.Len(t, chains[200], 1)
	assert.Equal(t, int32(100), chains[200][0].BlockerPid)
	assert.Equal(t, "ShareLock", chains[200][0].LockMode)
	assert.Equal(t, "orders", chains[200][0].RelationName)
}

func TestSessionQueryID(t *testing.T) {
	t.Parallel()

	withID := sessionRow{HasQueryID: true, QueryID: 42}
	assert.Equal(t, "42", sessionQueryID(withID))

	withoutID := sessionRow{Query: "SELECT 1"}
	assert.NotEmpty(t, sessionQueryID(withoutID))
}

func TestSessionDurationIdleInTransaction(t *testing.T) {
	t.Parallel()

	now := time.Now()
	session := sessionRow{
		State:            "idle in transaction",
		TransactionStart: now.Add(-45 * time.Second),
		QueryStart:       now.Add(-5 * time.Second),
	}

	duration := sessionDuration(session, now)
	assert.InDelta(t, 45.0, duration.Seconds(), 0.1)
}

func TestSessionDurationActiveQuery(t *testing.T) {
	t.Parallel()

	now := time.Now()
	session := sessionRow{
		State:      "active",
		QueryStart: now.Add(-12 * time.Second),
	}

	duration := sessionDuration(session, now)
	assert.InDelta(t, 12.0, duration.Seconds(), 0.1)
}

func TestQueryTextTruncation(t *testing.T) {
	t.Parallel()

	session := sessionRow{Query: "SELECT 1", TrackActivitySize: 8}
	assert.True(t, len(session.Query) >= int(session.TrackActivitySize))
	session.QueryTextTruncated = len(session.Query) >= int(session.TrackActivitySize)
	assert.True(t, session.QueryTextTruncated)
}
