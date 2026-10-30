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

package client

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func TestSplitQANCollectRequest(t *testing.T) {
	client := &Client{l: logrus.WithField("test", t.Name())}

	bucket := func(id string, queryLen int) *agentv1.MetricsBucket {
		query := strings.Repeat("A", queryLen)

		return &agentv1.MetricsBucket{
			Common: &agentv1.MetricsBucket_Common{
				Queryid:     id,
				Fingerprint: query,
				Example:     query,
			},
		}
	}

	// pmm-managed applies its MaxRecvMsgSize to the AgentMessage that wraps the request, not
	// to the request itself, so that is what has to be measured.
	assertFitsWire := func(t *testing.T, requests []*agentv1.QANCollectRequest) {
		t.Helper()

		for i, req := range requests {
			wire := proto.Size(&agentv1.AgentMessage{Id: 1, Payload: req.AgentMessageRequestPayload()})
			assert.LessOrEqualf(t, wire, maxAgentMessageSize,
				"AgentMessage %d of %d is %d bytes, over pmm-managed's %d byte limit: the server "+
					"fails the receive and resets the whole agent stream",
				i+1, len(requests), wire, maxAgentMessageSize)
		}
	}

	idsOf := func(requests []*agentv1.QANCollectRequest) []string {
		var ids []string
		for _, req := range requests {
			for _, b := range req.MetricsBucket {
				ids = append(ids, b.Common.Queryid)
			}
		}

		return ids
	}

	t.Run("An empty interval still produces exactly one request", func(t *testing.T) {
		requests := client.splitQANCollectRequest(&agentv1.QANCollectRequest{})
		require.Len(t, requests, 1)
		assert.Empty(t, requests[0].MetricsBucket)
	})

	t.Run("A small interval travels in one request", func(t *testing.T) {
		requests := client.splitQANCollectRequest(&agentv1.QANCollectRequest{
			MetricsBucket: []*agentv1.MetricsBucket{bucket("a", 8), bucket("b", 8)},
		})
		assertFitsWire(t, requests)
		require.Len(t, requests, 1)
		assert.Equal(t, []string{"a", "b"}, idsOf(requests))
	})

	t.Run("PMM-14976: a payload over the server limit is split, losing nothing", func(t *testing.T) {
		// 2669 buckets at --max-query-length=20480 is what reset the agent stream in the
		// live reproduction: 107890621 bytes against pmm-managed's 104857600 byte limit.
		const (
			bucketsN = 2669
			queryLen = 20480
		)
		buckets := make([]*agentv1.MetricsBucket, bucketsN)
		expected := make([]string, bucketsN)
		for i := range buckets {
			expected[i] = fmt.Sprintf("bucket %d", i)
			buckets[i] = bucket(expected[i], queryLen)
		}

		requests := client.splitQANCollectRequest(&agentv1.QANCollectRequest{MetricsBucket: buckets})

		assert.Greater(t, len(requests), 1, "a payload this large must be split")
		assertFitsWire(t, requests)

		// Splitting must not lose or reorder a single bucket.
		assert.Equal(t, expected, idsOf(requests))
	})

	t.Run("A bucket the server could never accept costs only itself", func(t *testing.T) {
		requests := client.splitQANCollectRequest(&agentv1.QANCollectRequest{
			MetricsBucket: []*agentv1.MetricsBucket{
				bucket("first", 8), bucket("undeliverable", maxQANCollectRequestSize), bucket("second", 8),
			},
		})
		assertFitsWire(t, requests)
		assert.Equal(t, []string{"first", "second"}, idsOf(requests))
	})

	t.Run("The largest bucket the filter keeps still fits on the wire", func(t *testing.T) {
		// A bucket just inside whatever the filter accepts is the worst case: it cannot be
		// batched with anything, so it travels alone and the envelope around it still has to
		// fit. Grow one field until the bucket sits as close under the filter as possible,
		// rather than hardcoding the encoding overhead.
		oneField := func(fingerprintLen int) *agentv1.MetricsBucket {
			return &agentv1.MetricsBucket{
				Common: &agentv1.MetricsBucket_Common{
					Queryid:     "probe",
					Fingerprint: strings.Repeat("A", fingerprintLen),
				},
			}
		}

		overhead := proto.Size(oneField(0)) + qanBucketOverhead
		biggest := oneField(maxAgentMessageSize - overhead)
		for proto.Size(biggest)+qanBucketOverhead > maxAgentMessageSize {
			overhead++
			biggest = oneField(maxAgentMessageSize - overhead)
		}
		require.LessOrEqual(t, proto.Size(biggest)+qanBucketOverhead, maxAgentMessageSize,
			"probe is meant to sit just inside the filter, not past it")

		assertFitsWire(t, client.splitQANCollectRequest(&agentv1.QANCollectRequest{
			MetricsBucket: []*agentv1.MetricsBucket{biggest},
		}))
	})
}

func TestNextQANBatch(t *testing.T) {
	sizesOf := func(n, each int) []int {
		sizes := make([]int, n)
		for i := range sizes {
			sizes[i] = each
		}

		return sizes
	}

	t.Run("Empty input yields an empty batch", func(t *testing.T) {
		assert.Equal(t, 0, nextQANBatch(nil))
	})

	t.Run("Buckets are capped by size", func(t *testing.T) {
		assert.Equal(t, 4, nextQANBatch(sizesOf(10, maxQANCollectRequestSize/4)))
	})

	t.Run("The batch stops before crossing the budget", func(t *testing.T) {
		assert.Equal(t, 2, nextQANBatch([]int{maxQANCollectRequestSize / 2, maxQANCollectRequestSize / 2, 1}))
	})

	t.Run("A bucket over the budget is sent on its own", func(t *testing.T) {
		assert.Equal(t, 1, nextQANBatch([]int{maxQANCollectRequestSize + 1, 8}))
	})
}
