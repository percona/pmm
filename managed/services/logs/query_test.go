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
		Start:       testStart,
		End:         testEnd,
		NodeNames:   []string{"node1", "node2"},
		Sources:     []string{"pmm-managed"},
		Presets:     []string{"postgres"},
		MinSeverity: 13,
		Text:        "deadlock",
		Pattern:     "connection ? from ?",
	})
	assert.Equal(t, testTimeWhere+
		" AND ServiceName IN ?"+
		" AND ResourceAttributes['node_name'] IN ?"+
		" AND LogAttributes['log.preset'] IN ?"+
		" AND SeverityNumber >= ?"+
		" AND positionCaseInsensitiveUTF8(Body, ?) > 0"+
		" AND "+PatternExpr("Body")+" = ?", where)
	assert.Equal(t, []any{
		testStart.Unix(), testEnd.Unix(), testStart.UnixNano(), testEnd.UnixNano(),
		[]string{"pmm-managed"},
		[]string{"node1", "node2"},
		[]string{"postgres"},
		int32(13),
		"deadlock",
		"connection ? from ?",
	}, args)
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
		assert.Contains(t, q, fmt.Sprintf("toStartOfInterval(Timestamp, INTERVAL %d SECOND)", int(expected.Seconds())), r)
		assert.Len(t, args, 4)
	}
	q, _, _ := volumeQuery(Filter{Start: testStart, End: testEnd})
	assert.Contains(t, q, "countIf(SeverityNumber >= 17) AS error")
	assert.Contains(t, q, "countIf(SeverityNumber >= 13 AND SeverityNumber < 17) AS warning")
	assert.Contains(t, q, "countIf(SeverityNumber >= 9 AND SeverityNumber < 13) AS info")
	assert.Contains(t, q, "countIf(SeverityNumber < 9) AS other")
}

func TestPatternsQuery_PreviousWindow(t *testing.T) {
	q, args := patternsQuery(Filter{Start: testStart, End: testEnd})
	prev := testStart.Add(-testEnd.Sub(testStart))
	require.Len(t, args, 5)
	assert.Equal(t, testStart.UnixNano(), args[0])
	assert.Equal(t, []any{prev.Unix(), testEnd.Unix(), prev.UnixNano(), testEnd.UnixNano()}, args[1:])
	assert.Contains(t, q, "HAVING current > 0 ORDER BY current DESC, p LIMIT 50")
	assert.Contains(t, q, "previous = 0 AS is_new")
}

func TestPatternExpr_Shared(t *testing.T) {
	expr := PatternExpr("Body")
	q, _ := patternsQuery(Filter{Start: testStart, End: testEnd})
	assert.Contains(t, q, expr+" AS p")
	where, _ := buildWhere(Filter{Start: testStart, End: testEnd, Pattern: "x"})
	assert.Contains(t, where, expr+" = ?")
	assert.True(t, strings.HasPrefix(expr, "substringUTF8("), expr)
	assert.True(t, strings.HasSuffix(expr, ", 1, 200)"), expr)
}

func TestContextQuery(t *testing.T) {
	at := time.Date(2026, 10, 17, 10, 30, 0, 123456789, time.UTC)
	q, args := contextQuery(at, "node1", "/var/log/postgresql/postgresql.log", 20)
	assert.Contains(t, q, "substringUTF8(Body, 1, 65536)")
	assert.Contains(t, q, "Timestamp <= fromUnixTimestamp64Nano(?)")
	assert.Contains(t, q, "Timestamp > fromUnixTimestamp64Nano(?)")
	assert.Contains(t, args, "node1")
	assert.Contains(t, args, "/var/log/postgresql/postgresql.log")
	assert.Contains(t, args, at.UnixNano())
	assert.Equal(t, 20, args[len(args)-1])
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
	context, _ := contextQuery(testStart, "n", "f", 5)
	nodes, sources, presets, _ := facetsQueries(f)
	for _, q := range []string{search, volume, patterns, context, nodes, sources, presets} {
		assert.True(t, strings.HasSuffix(q, " SETTINGS max_execution_time = 30, readonly = 1"), q)
	}
}
