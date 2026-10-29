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
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const (
	// Largest AgentMessage pmm-managed accepts, mirroring the MaxRecvMsgSize enforced
	// there. Going over it does not merely drop the payload: the server fails the receive
	// and the whole agent stream is reset, taking metrics, actions and status down with it.
	maxAgentMessageSize = 100 * 1024 * 1024

	// Budget for the buckets in one QANCollectRequest. It is also the point at which a
	// bucket is given up on, so that anything kept provably still fits once wrapped: the
	// limit above applies to the AgentMessage envelope, not to the request inside it.
	maxQANCollectRequestSize = 80 * 1024 * 1024

	// Upper bound on what one bucket costs on top of its own encoded size: metrics_bucket
	// is field 1 of QANCollectRequest, so one tag byte plus its length varint.
	qanBucketOverhead = 5
)

// Splits one collection interval's buckets into QANCollectRequests that each stay within
// what pmm-managed accepts. A bucket too large to travel even on its own is dropped: the
// server would fail the receive and reset the agent stream, which costs every other bucket
// in the interval plus the connection itself.
func (c *Client) splitQANCollectRequest(collect *agentv1.QANCollectRequest) []*agentv1.QANCollectRequest {
	// Rebuilding the request carries over metrics_bucket and nothing else, which is the
	// whole of QANCollectRequest today. A field added to it later has to be copied here too.
	buckets := collect.MetricsBucket

	// Unlike the pmm-managed side, which compacts its own freshly built slice in place, this
	// slice was built by a QAN collector and handed over a channel, so it is not ours to
	// modify. Copying costs one pointer per bucket against a payload of tens of MB.
	kept := make([]*agentv1.MetricsBucket, 0, len(buckets))
	sizes := make([]int, 0, len(buckets))

	for _, b := range buckets {
		size := proto.Size(b) + qanBucketOverhead
		if size > maxQANCollectRequestSize {
			c.l.Warnf("Dropping metrics bucket for query '%s': %d bytes exceeds the %d byte "+
				"budget for one request to PMM Server. Set a max_query_length for that service.",
				b.Common.GetQueryid(), size, maxQANCollectRequestSize)

			continue
		}

		kept = append(kept, b)
		sizes = append(sizes, size)
	}

	// Send at least one request, even an empty one, so the server keeps seeing exactly one
	// request per interval in the common case.
	var requests []*agentv1.QANCollectRequest
	for from := 0; ; {
		n := nextQANBatch(sizes[from:])
		// Capped subslice: an append by a later caller must not reach into the next chunk.
		requests = append(requests, &agentv1.QANCollectRequest{MetricsBucket: kept[from : from+n : from+n]})

		from += n
		if from >= len(kept) {
			return requests
		}
	}
}

// Returns how many buckets from the front of sizes fit into one QANCollectRequest. Never 0
// for a non-empty slice, so splitQANCollectRequest always makes progress.
func nextQANBatch(sizes []int) int {
	var total int
	for i, size := range sizes {
		total += size
		if total > maxQANCollectRequestSize {
			if i == 0 {
				return 1
			}

			return i
		}
	}

	return len(sizes)
}
