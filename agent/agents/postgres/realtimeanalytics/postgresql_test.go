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
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/percona/pmm/agent/agents"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

func TestActivityQueryIsTagged(t *testing.T) {
	t.Parallel()

	assert.True(t, strings.HasPrefix(activityQuery, "WITH "+agents.RTAQueryTag+" "))
}

func TestNewCollectInterval(t *testing.T) {
	t.Parallel()

	l := logrus.NewEntry(logrus.New())
	for _, interval := range []time.Duration{0, -1} {
		m, err := New(&Params{CollectInterval: interval}, l)
		require.NoError(t, err)
		assert.Equal(t, defaultCollectInterval, m.collectInterval)
	}
}

func TestInstanceAddress(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "pg-1:5432", instanceAddress("postgres://pmm:secret@pg-1:5432/postgres?sslmode=disable"))
	assert.Equal(t, "[::1]:5433", instanceAddress("postgres://pmm@[::1]:5433/postgres"))
	assert.Equal(t, "/var/run/postgresql", instanceAddress("postgres://pmm:secret@/postgres?host=%2Fvar%2Frun%2Fpostgresql&sslmode=disable"))
	assert.Empty(t, instanceAddress("postgres://%zz"))
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

func TestWithBlockingChains(t *testing.T) {
	t.Parallel()

	session := func(pid int32, blockers ...*rtav1.BlockingTransaction) *rtav1.QueryData {
		return &rtav1.QueryData{Payload: &rtav1.QueryData_PostgresqlPayload{
			PostgresqlPayload: &rtav1.QueryPostgreSQLData{Pid: pid, BlockedBy: blockers},
		}}
	}
	blocker := func(pid int64, root bool, waited time.Duration) *rtav1.BlockingTransaction {
		return &rtav1.BlockingTransaction{BlockingConnId: pid, Root: root, WaitDuration: durationpb.New(waited)}
	}

	// 7 is idle in transaction; 8 waits for 7, 9 waits for 8.
	queries := []*rtav1.QueryData{
		session(7),
		session(8, blocker(7, true, 10*time.Second)),
		session(9, blocker(8, false, 5*time.Second)),
	}
	withBlockingChains(queries)

	assert.Empty(t, queries[0].GetPostgresqlPayload().BlockedBy)
	assert.Len(t, queries[1].GetPostgresqlPayload().BlockedBy, 1)

	chain := queries[2].GetPostgresqlPayload().BlockedBy
	require.Len(t, chain, 2)
	assert.Equal(t, int64(7), chain[0].BlockingConnId)
	assert.True(t, chain[0].Root)
	assert.Equal(t, 5*time.Second, chain[0].WaitDuration.AsDuration(), "the waiter's own wait")
	assert.Equal(t, int64(8), chain[1].BlockingConnId)
	assert.False(t, chain[1].Root)

	// the shared blocker of session 8 is not modified
	assert.Equal(t, 10*time.Second, queries[1].GetPostgresqlPayload().BlockedBy[0].WaitDuration.AsDuration())

	// A deadlock not yet broken by deadlock_timeout: neither session lists itself.
	cycle := []*rtav1.QueryData{
		session(5, blocker(6, false, time.Second)),
		session(6, blocker(5, false, time.Second)),
	}
	withBlockingChains(cycle)

	for i, pid := range []int64{6, 5} {
		chain := cycle[i].GetPostgresqlPayload().BlockedBy
		require.Len(t, chain, 1)
		assert.Equal(t, pid, chain[0].BlockingConnId)
	}
}
