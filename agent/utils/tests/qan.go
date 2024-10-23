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

package tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// AssertBucketsEqual asserts that two MetricsBuckets are equal while providing a good diff.
func AssertBucketsEqual(t *testing.T, expected, actual *agentv1.MetricsBucket) bool {
	t.Helper()

	return assert.Equal(t, prototext.Format(expected), prototext.Format(actual))
}

// FormatBuckets formats MetricsBuckets to string for tests.
func FormatBuckets(mb []*agentv1.MetricsBucket) string {
	res := make([]string, len(mb))
	for i, b := range mb {
		res[i] = prototext.Format(b)
	}
	return strings.Join(res, "\n")
}
