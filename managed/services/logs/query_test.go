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

package logs

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/otel"
)

var (
	testStart = time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	testEnd   = time.Date(2026, 10, 17, 11, 0, 0, 500, time.UTC)
)

const testTimeWhere = "TimestampTime BETWEEN toDateTime(?) AND toDateTime(?) AND " +
	"Timestamp BETWEEN fromUnixTimestamp64Nano(?) AND fromUnixTimestamp64Nano(?)"

func TestBuildWhere_Empty(t *testing.T) {
	where, args := buildWhere(Filter{Start: testStart, End: testEnd})
	assert.Equal(t, testTimeWhere, where)
	assert.Equal(t, []any{testStart.Unix(), testEnd.Unix(), testStart.UnixNano(), testEnd.UnixNano()}, args)
}

func TestBuildWhere_All(t *testing.T) {
	where, args := buildWhere(Filter{
		Start:            testStart,
		End:              testEnd,
		NodeNames:        []string{"node1", "node2"},
		ExcludeNodeNames: []string{"pmm-server"},
		Sources:          []string{"pmm-managed"},
		Presets:          []string{"postgres"},
		MinSeverity:      13,
		MaxSeverity:      16,
		Text:             "deadlock",
		Pattern:          "connection ? from ?",
	})
	assert.Equal(t, testTimeWhere+
		" AND ServiceName IN ?"+
		" AND ResourceAttributes['node_name'] IN ?"+
		" AND LogAttributes['log.preset'] IN ?"+
		" AND ResourceAttributes['node_name'] NOT IN ?"+
		" AND SeverityNumber >= ?"+
		" AND SeverityNumber <= ?"+
		" AND positionCaseInsensitiveUTF8(Body, ?) > 0"+
		" AND PatternHash = cityHash64(?)", where)
	assert.Equal(t, []any{
		testStart.Unix(), testEnd.Unix(), testStart.UnixNano(), testEnd.UnixNano(),
		[]string{"pmm-managed"},
		[]string{"node1", "node2"},
		[]string{"postgres"},
		[]string{"pmm-server"},
		int32(13),
		int32(16),
		"deadlock",
		"connection ? from ?",
	}, args)
}

func TestBuildWhere_PatternHashes(t *testing.T) {
	where, args := buildWhere(Filter{Start: testStart, End: testEnd, PatternHash: 42, ExcludePatternHashes: []uint64{7, 9}})
	assert.Equal(t, testTimeWhere+" AND PatternHash = ? AND PatternHash NOT IN ?", where)
	assert.Equal(t, []any{uint64(42), []uint64{7, 9}}, args[len(args)-2:])
}

func TestLineColumns_PatternHash(t *testing.T) {
	assert.True(t, strings.HasSuffix(lineColumns(10), ", PatternHash"), lineColumns(10))
	assert.True(t, strings.HasSuffix(lineColumnNames, ", PatternHash"), lineColumnNames)
}

func TestPatternsQuery_TopSources(t *testing.T) {
	q, _ := patternsQuery(Filter{Start: testStart, End: testEnd})
	assert.Contains(t, q, "SELECT "+otel.PatternExpr("body")+" AS p, PatternHash, current, previous, ")
	assert.Contains(t, q, " AS sample, CAST([], 'Array(String)') AS top_sources FROM (")
	for _, f := range []Filter{
		{Start: testStart, End: testEnd, PatternHash: 42},
		{Start: testStart, End: testEnd, Pattern: "x"},
	} {
		q, _ = patternsQuery(f)
		assert.Contains(t, q, " AS sample, top_sources FROM (")
		assert.Contains(t, q, "topKIf(3)(ServiceName, cur) AS top_sources FROM otel.logs")
	}
}

func TestBuildWhere_TextLiteral(t *testing.T) {
	text := `50%_off "x" \d+`
	where, args := buildWhere(Filter{Start: testStart, End: testEnd, Text: text})
	assert.NotContains(t, where, "50%")
	assert.NotContains(t, where, `\d+`)
	assert.Equal(t, text, args[len(args)-1])
}

func TestSearchQuery_Cursor(t *testing.T) {
	c := Cursor{Time: time.Date(2026, 10, 17, 10, 30, 0, 123456789, time.UTC), Hash: 18446744073709551615, Skip: 3}
	decoded, err := DecodeCursor(c.Encode())
	require.NoError(t, err)
	assert.True(t, c.Time.Equal(decoded.Time))
	assert.Equal(t, c.Hash, decoded.Hash)
	assert.Equal(t, 3, decoded.Skip)

	// Lines identical in every shown field share (Timestamp, h): the cursor includes them and skips those returned.
	q, args := searchQuery(Filter{Start: testStart, End: testEnd}, &c, 100)
	assert.Contains(t, q, "AND (Timestamp, h) <= (fromUnixTimestamp64Nano(?), ?)")
	assert.True(t, strings.HasSuffix(q, " LIMIT ? OFFSET ?"+querySettings), q)
	assert.Equal(t, []any{c.Time.UnixNano(), c.Hash, 100, 3}, args[len(args)-4:])

	q, _ = searchQuery(Filter{Start: testStart, End: testEnd}, nil, 100)
	assert.NotContains(t, q, "(Timestamp, h)")
	assert.NotContains(t, q, "OFFSET")

	for _, s := range []string{"", "not base64!", c.Encode()[:3], "MTIz", "MS4yLng"} {
		_, err = DecodeCursor(s)
		assert.Error(t, err, s)
	}
}

func TestSearchQuery_TopNCutoff(t *testing.T) {
	f := Filter{Start: testStart, End: testEnd, Sources: []string{"pmm-managed"}}
	where, whereArgs := buildWhere(f)

	q, args := searchQuery(f, nil, 50)
	assert.Contains(t, q, "WHERE "+where+" AND Timestamp >= (SELECT min(Timestamp) FROM "+
		"(SELECT Timestamp FROM otel.logs WHERE "+where+" ORDER BY Timestamp DESC LIMIT ?)) ORDER BY")
	expected := append(append(append([]any{}, whereArgs...), whereArgs...), 50, 50)
	assert.Equal(t, expected, args)

	c := Cursor{Time: testStart.Add(time.Minute), Hash: 7}
	q, args = searchQuery(f, &c, 50)
	assert.Contains(t, q, "(SELECT Timestamp FROM otel.logs WHERE "+where+
		" AND Timestamp < fromUnixTimestamp64Nano(?) ORDER BY Timestamp DESC LIMIT ?)")
	expected = append(append(append([]any{}, whereArgs...), whereArgs...), c.Time.UnixNano(), 50, c.Time.UnixNano(), c.Hash, 50, 0)
	assert.Equal(t, expected, args)
}

func TestSearchQuery_Truncates(t *testing.T) {
	q, _ := searchQuery(Filter{Start: testStart, End: testEnd}, nil, 0)
	assert.Contains(t, q, "substringUTF8(Body, 1, 4096)")
	assert.Contains(t, q, "lengthUTF8(Body) > 4096")
	assert.Contains(t, q, "cityHash64(Body, ServiceName, ResourceAttributes['node_name'], LogAttributes, SeverityNumber, SeverityText) AS h")
	assert.Contains(t, q, "ORDER BY Timestamp DESC, h DESC LIMIT ?")
}

func TestSearchQuery_Limit(t *testing.T) {
	for limit, expected := range map[int]int{0: 200, -1: 200, 50: 50, 500: 500, 1000: 500} {
		_, args := searchQuery(Filter{Start: testStart, End: testEnd}, nil, limit)
		assert.Equal(t, expected, args[len(args)-1], limit)
	}
}

func TestVolumeQuery_Step(t *testing.T) {
	for r, expected := range map[time.Duration]time.Duration{
		10 * time.Second:    time.Second,
		10 * time.Minute:    10 * time.Second,
		time.Hour:           time.Minute,
		24 * time.Hour:      30 * time.Minute,
		7 * 24 * time.Hour:  3 * time.Hour,
		30 * 24 * time.Hour: 12 * time.Hour,
		1000 * time.Hour:    12 * time.Hour,
		3000 * time.Hour:    24 * time.Hour,
	} {
		q, args, step := volumeQuery(Filter{Start: testStart, End: testStart.Add(r)})
		assert.Equal(t, expected, step, r)
		n := int(expected.Seconds())
		assert.Contains(t, q, fmt.Sprintf("toStartOfInterval(Timestamp, INTERVAL %d SECOND)", n), r)
		// Empty buckets are returned too, so the chart's time axis is linear; at most 100 of them.
		assert.Contains(t, q, fmt.Sprintf("ORDER BY t WITH FILL FROM toStartOfInterval(toDateTime(?), INTERVAL %d SECOND) "+
			"TO toDateTime(?) + 1 STEP INTERVAL %d SECOND", n, n), r)
		assert.Equal(t, []any{testStart.Unix(), testStart.Add(r).Unix()}, args[4:], r)
		if r <= 99*24*time.Hour {
			assert.LessOrEqual(t, int(r/expected)+1, 100, r)
		}
	}
	q, _, _ := volumeQuery(Filter{Start: testStart, End: testEnd})
	assert.Contains(t, q, "countIf(SeverityNumber >= 17) AS error")
	assert.Contains(t, q, "countIf(SeverityNumber >= 13 AND SeverityNumber < 17) AS warning")
	assert.Contains(t, q, "countIf(SeverityNumber >= 9 AND SeverityNumber < 13) AS info")
	assert.Contains(t, q, "countIf(SeverityNumber < 9) AS other")
}

func TestPatternsQuery_PreviousWindow(t *testing.T) {
	q, args := patternsQuery(Filter{Start: testStart, End: testEnd, NodeNames: []string{"n1"}, Text: "x", MinSeverity: 17})
	prev := testStart.Add(-testEnd.Sub(testStart))
	// The baseline asks whether the scope had any line before the range; search and severity do not narrow it,
	// or a search for a new message would never find it new.
	baseWhere, baseArgs := buildWhere(Filter{Start: prev, End: testStart, NodeNames: []string{"n1"}})
	assert.Contains(t, q, "previous = 0 AND (SELECT count() FROM (SELECT 1 FROM otel.logs WHERE "+baseWhere+" LIMIT 1)) > 0 AS is_new")
	// The main query's arguments: four time bounds, the node list, the severity and the search text.
	require.Len(t, args, len(baseArgs)+1+7)
	assert.Equal(t, baseArgs, args[:len(baseArgs)])
	assert.Equal(t, testStart.UnixNano(), args[len(baseArgs)])
	assert.Equal(t, []any{prev.Unix(), testEnd.Unix(), prev.UnixNano(), testEnd.UnixNano()}, args[len(baseArgs)+1:len(baseArgs)+5])
	assert.Contains(t, q, ") WHERE current > 0 ORDER BY current DESC, p LIMIT 50")
}

func TestPatterns_UseStoredHash(t *testing.T) {
	q, _ := patternsQuery(Filter{Start: testStart, End: testEnd})
	assert.Contains(t, q, "GROUP BY PatternHash")
	// Only the returned patterns are masked, from a sample line, instead of every line in the range.
	assert.Contains(t, q, otel.PatternExpr("body")+" AS p")
	assert.NotContains(t, q, otel.PatternExpr("Body"))
	where, args := buildWhere(Filter{Start: testStart, End: testEnd, Pattern: "x"})
	assert.True(t, strings.HasSuffix(where, " AND PatternHash = cityHash64(?)"), where)
	assert.Equal(t, "x", args[len(args)-1])
	// clickhouse-go switches to {name:Type} query parameters, and rejects our ? arguments, when the query matches this.
	assert.NotRegexp(t, "{.+:.+}", q)
}

func TestContextQuery(t *testing.T) {
	at := time.Date(2026, 10, 17, 10, 30, 0, 123456789, time.UTC)
	q, args := contextQuery(at, "node1", "/var/log/postgresql/postgresql.log", "src", 20)
	assert.Contains(t, q, "substringUTF8(Body, 1, 65536)")
	// Every line at the anchor's own timestamp, so the clicked line is always among them.
	assert.Contains(t, q, "Timestamp < fromUnixTimestamp64Nano(?) ORDER BY Timestamp DESC LIMIT ?")
	assert.Contains(t, q, "Timestamp = fromUnixTimestamp64Nano(?) LIMIT 100")
	assert.Contains(t, q, "Timestamp > fromUnixTimestamp64Nano(?) ORDER BY Timestamp LIMIT ?")
	assert.Equal(t, 3, strings.Count(q, "ServiceName = ? AND "))
	assert.Contains(t, args, "src")
	assert.Contains(t, args, "node1")
	assert.Contains(t, args, "/var/log/postgresql/postgresql.log")
	assert.Contains(t, args, at.UnixNano())
	assert.Equal(t, 20, args[len(args)-1])

	q, args = contextQuery(at, "node1", "f", "", 20)
	assert.NotContains(t, q, "ServiceName = ?")
	assert.NotContains(t, args, "")
}

func TestFacetsQueries(t *testing.T) {
	nodes, sources, presets, args := facetsQueries(Filter{Start: testStart, End: testEnd, Sources: []string{"x"}, Text: "y"})
	assert.Len(t, args, 4)
	for _, q := range []string{nodes, sources, presets} {
		assert.Contains(t, q, "WHERE "+testTimeWhere+" GROUP BY v HAVING v != '' ORDER BY count() DESC LIMIT 200")
	}
	assert.Contains(t, nodes, "ResourceAttributes['node_name'] AS v")
	assert.Contains(t, sources, "ServiceName AS v")
	assert.Contains(t, presets, "LogAttributes['log.preset'] AS v")
}

func TestQueries_Settings(t *testing.T) {
	f := Filter{Start: testStart, End: testEnd}
	search, _ := searchQuery(f, nil, 0)
	volume, _, _ := volumeQuery(f)
	patterns, _ := patternsQuery(f)
	context, _ := contextQuery(testStart, "n", "f", "s", 5)
	nodes, sources, presets, _ := facetsQueries(f)
	for _, q := range []string{search, volume, patterns, context, nodes, sources, presets} {
		assert.True(t, strings.HasSuffix(q, " SETTINGS max_execution_time = 30, readonly = 1"), q)
	}
}
