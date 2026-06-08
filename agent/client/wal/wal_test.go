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

package wal

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func req(qid string) *agentv1.QANCollectRequest {
	return &agentv1.QANCollectRequest{
		MetricsBucket: []*agentv1.MetricsBucket{{Common: &agentv1.MetricsBucket_Common{Queryid: qid}}},
	}
}

func TestSpoolFIFOAndPersistence(t *testing.T) {
	t.Parallel()
	l := logrus.WithField("test", t.Name())
	dir := t.TempDir()

	s, err := New(dir, 1<<20, l)
	require.NoError(t, err)
	require.NoError(t, s.Enqueue(req("a")))
	require.NoError(t, s.Enqueue(req("b")))
	require.Equal(t, 2, s.Len())

	// FIFO: oldest first, Next does not remove.
	it, ok := s.Next()
	require.True(t, ok)
	require.Equal(t, "a", it.Request.MetricsBucket[0].Common.Queryid)
	require.Equal(t, 2, s.Len())

	// Reopen without acking: both entries replay (restart durability).
	s2, err := New(dir, 1<<20, l)
	require.NoError(t, err)
	require.Equal(t, 2, s2.Len())

	// Ack the oldest, reopen: only "b" survives.
	it2, ok := s2.Next()
	require.True(t, ok)
	s2.Remove(it2.Seq)
	require.Equal(t, 1, s2.Len())

	s3, err := New(dir, 1<<20, l)
	require.NoError(t, err)
	it3, ok := s3.Next()
	require.True(t, ok)
	require.Equal(t, "b", it3.Request.MetricsBucket[0].Common.Queryid)
}

func TestSpoolDropOldestOverCapacity(t *testing.T) {
	t.Parallel()
	l := logrus.WithField("test", t.Name())

	one, err := proto.Marshal(req("a"))
	require.NoError(t, err)

	// Cap leaves room for ~one entry, so each new enqueue evicts the oldest.
	s, err := New(t.TempDir(), int64(len(one)+1), l)
	require.NoError(t, err)
	require.NoError(t, s.Enqueue(req("a")))
	require.NoError(t, s.Enqueue(req("b")))
	require.NoError(t, s.Enqueue(req("c")))

	require.Equal(t, 1, s.Len())
	require.GreaterOrEqual(t, s.Dropped(), uint64(2))

	it, ok := s.Next()
	require.True(t, ok)
	require.Equal(t, "c", it.Request.MetricsBucket[0].Common.Queryid) // newest kept
}

func TestSpoolEmpty(t *testing.T) {
	t.Parallel()
	s, err := New(t.TempDir(), 1<<20, logrus.WithField("test", t.Name()))
	require.NoError(t, err)
	_, ok := s.Next()
	require.False(t, ok)
	require.Zero(t, s.Len())
}

// TestSpoolConcurrent runs the single producer (Enqueue) and single consumer
// (Next/Remove) concurrently; run under -race it proves disk I/O happens off the
// lock without a data race, and that every entry is delivered and drained.
func TestSpoolConcurrent(t *testing.T) {
	t.Parallel()
	s, err := New(t.TempDir(), 1<<20, logrus.WithField("test", t.Name()))
	require.NoError(t, err)

	const n = 200
	done := make(chan struct{})
	go func() { // consumer
		defer close(done)
		for got := 0; got < n; {
			item, ok := s.Next()
			if !ok {
				select {
				case <-s.Notify():
				case <-time.After(time.Second):
				}
				continue
			}
			s.Remove(item.Seq)
			got++
		}
	}()

	for i := range n { // producer
		require.NoError(t, s.Enqueue(req(fmt.Sprintf("q%d", i))))
	}

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("consumer did not drain the spool")
	}
	require.Zero(t, s.Len())
}
