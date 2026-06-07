// Copyright (C) 2023 Percona LLC
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

package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
)

func TestResultCache(t *testing.T) {
	t.Parallel()
	c := newResultCache(50 * time.Millisecond)

	_, ok := c.get("k")
	require.False(t, ok)

	c.set("k", "v")
	got, ok := c.get("k")
	require.True(t, ok)
	require.Equal(t, "v", got)

	time.Sleep(60 * time.Millisecond)
	_, ok = c.get("k")
	require.False(t, ok, "entry should expire after the TTL")
}

func TestCacheKey(t *testing.T) {
	t.Parallel()
	a := &qanv1.GetReportRequest{GroupBy: "queryid", Limit: 10}
	b := &qanv1.GetReportRequest{GroupBy: "queryid", Limit: 10}
	c := &qanv1.GetReportRequest{GroupBy: "service_name", Limit: 10}

	require.NotEmpty(t, cacheKey("report", a))
	require.Equal(t, cacheKey("report", a), cacheKey("report", b), "identical requests share a key")
	require.NotEqual(t, cacheKey("report", a), cacheKey("report", c), "different requests differ")
}
