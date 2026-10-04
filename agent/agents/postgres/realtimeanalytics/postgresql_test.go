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

	"github.com/percona/pmm/agent/agents"
)

func TestActivityQueryIsTagged(t *testing.T) {
	t.Parallel()

	assert.True(t, agents.IsRTAQuery(activityQuery))
}

func TestToBlockers(t *testing.T) {
	t.Parallel()

	t.Run("NoBlockers", func(t *testing.T) {
		t.Parallel()

		res, err := toBlockers(nil, nil)
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Chain", func(t *testing.T) {
		t.Parallel()

		waited := 2.5
		raw := []byte(`[{"pid": 7, "query": "UPDATE t SET v = 1", "state": "idle in transaction", "user": "app",
			"xact_secs": 10.25, "root": true, "truncated": false},
			{"pid": 8, "query": "UPDATE t SET v = 2", "state": "active", "user": "app",
			"xact_secs": null, "root": false, "truncated": true}]`)

		res, err := toBlockers(raw, &waited)
		require.NoError(t, err)
		require.Len(t, res, 2)

		assert.Equal(t, int64(7), res[0].BlockingConnId)
		assert.Equal(t, "UPDATE t SET v = 1", res[0].BlockingQuery)
		assert.Equal(t, "idle in transaction", res[0].BlockingCommand)
		assert.Equal(t, "app", res[0].BlockingUsername)
		assert.Equal(t, 2500*time.Millisecond, res[0].WaitDuration.AsDuration())
		assert.Equal(t, 10250*time.Millisecond, res[0].BlockerTransactionDuration.AsDuration())
		assert.True(t, res[0].Root)
		assert.False(t, res[0].BlockingQueryTruncated)

		assert.Nil(t, res[1].BlockerTransactionDuration)
		assert.False(t, res[1].Root)
		assert.True(t, res[1].BlockingQueryTruncated)
	})

	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()

		_, err := toBlockers([]byte(`{`), nil)
		require.Error(t, err)
	})
}
