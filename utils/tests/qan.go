// pmm-agent
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package tests

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/percona/pmm/api/agentpb"
	"github.com/stretchr/testify/assert"
)

// AssertBucketsEqual asserts that two MetricsBuckets are equal while providing a good diff.
func AssertBucketsEqual(t *testing.T, expected, actual *agentpb.MetricsBucket) bool {
	t.Helper()

	return assert.Equal(t, proto.MarshalTextString(expected), proto.MarshalTextString(actual))
}

// FormatBuckets formats MetricsBuckets to string for tests.
func FormatBuckets(mb []*agentpb.MetricsBucket) string {
	res := make([]string, len(mb))
	for i, b := range mb {
		res[i] = proto.MarshalTextString(b)
	}
	return strings.Join(res, "\n")
}
