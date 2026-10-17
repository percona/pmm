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

// Package logs serves log lines, volume, patterns and facets from ClickHouse otel.logs.
package logs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSearchLimit = 200
	maxSearchLimit     = 500
	listBodyLimit      = 4096
	contextBodyLimit   = 65536
	contextWindow      = 24 * time.Hour
	maxVolumeBuckets   = 100
	querySettings      = " SETTINGS max_execution_time = 30, readonly = 1"
)

// volumeSteps are the volume bucket widths, smallest first.
var volumeSteps = []time.Duration{
	time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second,
	time.Minute, 5 * time.Minute, 10 * time.Minute, 30 * time.Minute,
	time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
}

// Filter is the shared filter of every logs call.
type Filter struct {
	Start, End  time.Time
	NodeNames   []string
	Sources     []string
	Presets     []string
	MinSeverity int32
	Text        string
	Pattern     string
}

// Cursor positions a search page after the last returned line.
type Cursor struct {
	Time time.Time
	Hash uint64
}

// Encode returns the opaque form of c that clients pass back.
func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(c.Time.UnixNano(), 10) + "." + strconv.FormatUint(c.Hash, 10)))
}

// DecodeCursor parses a cursor made by Cursor.Encode.
func DecodeCursor(s string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	nanos, hash, ok := strings.Cut(string(b), ".")
	if !ok {
		return Cursor{}, errors.New("invalid cursor")
	}
	n, err := strconv.ParseInt(nanos, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	h, err := strconv.ParseUint(hash, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	return Cursor{Time: time.Unix(0, n).UTC(), Hash: h}, nil
}

// PatternExpr returns the ClickHouse expression that turns column into its pattern; GetLogPatterns and the pattern filter both use it.
func PatternExpr(column string) string {
	expr := column
	for _, r := range [][2]string{
		{`'[^']*'|"[^"]*"`, `'?'`},
		{`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, `?`},
		{`\b[0-9a-fA-F]{8,}\b`, `?`},
		{`\b[0-9]{1,3}(\.[0-9]{1,3}){3}\b`, `?`},
		{`[0-9]+(\.[0-9]+)?`, `?`},
	} {
		expr = "replaceRegexpAll(" + expr + ", " + sqlString(r[0]) + ", " + sqlString(r[1]) + ")"
	}
	return "substringUTF8(" + expr + ", 1, 200)"
}

// sqlString quotes a constant as a ClickHouse string literal.
func sqlString(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

// timeWhere returns the conditions selecting [start, end]; TimestampTime prunes by the primary key and partition.
func timeWhere(start, end time.Time) (string, []any) {
	return "TimestampTime BETWEEN toDateTime(?) AND toDateTime(?) AND " +
			"Timestamp BETWEEN fromUnixTimestamp64Nano(?) AND fromUnixTimestamp64Nano(?)",
		[]any{start.Unix(), end.Unix(), start.UnixNano(), end.UnixNano()}
}

func buildWhere(f Filter) (string, []any) {
	where, args := timeWhere(f.Start, f.End)
	conds := []string{where}
	for _, in := range []struct {
		column string
		values []string
	}{
		{"ServiceName", f.Sources},
		{"ResourceAttributes['node_name']", f.NodeNames},
		{"LogAttributes['log.preset']", f.Presets},
	} {
		if len(in.values) > 0 {
			conds = append(conds, in.column+" IN ?")
			args = append(args, in.values)
		}
	}
	if f.MinSeverity > 0 {
		conds = append(conds, "SeverityNumber >= ?")
		args = append(args, f.MinSeverity)
	}
	if f.Text != "" {
		conds = append(conds, "positionCaseInsensitiveUTF8(Body, ?) > 0")
		args = append(args, f.Text)
	}
	if f.Pattern != "" {
		conds = append(conds, PatternExpr("Body")+" = ?")
		args = append(args, f.Pattern)
	}
	return strings.Join(conds, " AND "), args
}

// lineColumnNames are the names lineColumns gives the columns of a log line.
const lineColumnNames = "Timestamp, node_name, ServiceName, preset, file_name, SeverityNumber, SeverityText, body, body_truncated, LogAttributes"

// lineColumns returns the columns of a log line with the body cut to bodyLimit characters.
func lineColumns(bodyLimit int) string {
	return fmt.Sprintf("Timestamp, ResourceAttributes['node_name'] AS node_name, ServiceName, LogAttributes['log.preset'] AS preset, "+
		"LogAttributes['log.file.name'] AS file_name, SeverityNumber, SeverityText, substringUTF8(Body, 1, %d) AS body, "+
		"lengthUTF8(Body) > %d AS body_truncated, LogAttributes", bodyLimit, bodyLimit)
}

// pageSize returns the search page size for a requested limit.
func pageSize(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	return min(limit, maxSearchLimit)
}

// searchQuery returns the newest page of lines older than after; its subquery bounds Timestamp first, so only the page's rows are hashed and sorted.
func searchQuery(f Filter, after *Cursor, limit int) (string, []any) {
	size := pageSize(limit)
	where, whereArgs := buildWhere(f)
	inner, innerArgs := where, append([]any{}, whereArgs...)
	if after != nil {
		// Strictly older, so lines sharing after's timestamp stay above the bound; the tuple comparison drops returned ones.
		inner += " AND Timestamp < fromUnixTimestamp64Nano(?)"
		innerArgs = append(innerArgs, after.Time.UnixNano())
	}
	cond := where + " AND Timestamp >= (SELECT min(Timestamp) FROM (SELECT Timestamp FROM otel.logs WHERE " + inner +
		" ORDER BY Timestamp DESC LIMIT ?))"
	args := append(append(append([]any{}, whereArgs...), innerArgs...), size)
	if after != nil {
		cond += " AND (Timestamp, h) < (fromUnixTimestamp64Nano(?), ?)"
		args = append(args, after.Time.UnixNano(), after.Hash)
	}
	q := "SELECT " + lineColumns(listBodyLimit) + ", cityHash64(Body, ServiceName, LogAttributes['log.file.name']) AS h" +
		" FROM otel.logs WHERE " + cond + " ORDER BY Timestamp DESC, h DESC LIMIT ?" + querySettings
	return q, append(args, size)
}

func volumeQuery(f Filter) (string, []any, time.Duration) {
	r := f.End.Sub(f.Start)
	step := volumeSteps[len(volumeSteps)-1]
	for _, s := range volumeSteps {
		if r <= s*maxVolumeBuckets {
			step = s
			break
		}
	}
	where, args := buildWhere(f)
	q := fmt.Sprintf("SELECT toStartOfInterval(Timestamp, INTERVAL %d SECOND) AS t, "+
		"countIf(SeverityNumber >= 17) AS error, "+
		"countIf(SeverityNumber >= 13 AND SeverityNumber < 17) AS warning, "+
		"countIf(SeverityNumber >= 9 AND SeverityNumber < 13) AS info, "+
		"countIf(SeverityNumber < 9) AS other "+
		"FROM otel.logs WHERE %s GROUP BY t ORDER BY t", int(step.Seconds()), where)
	return q + querySettings, args, step
}

func patternsQuery(f Filter) (string, []any) {
	prev := f
	prev.Start = f.Start.Add(-f.End.Sub(f.Start))
	where, args := buildWhere(prev)
	q := fmt.Sprintf("WITH Timestamp >= fromUnixTimestamp64Nano(?) AS cur SELECT %s AS p, "+
		"countIf(cur) AS current, countIf(NOT cur) AS previous, previous = 0 AS is_new, "+
		"maxIf(SeverityNumber, cur) AS max_severity, minIf(Timestamp, cur) AS first_seen, maxIf(Timestamp, cur) AS last_seen, "+
		"substringUTF8(anyIf(Body, cur), 1, %d) AS sample "+
		"FROM otel.logs WHERE %s GROUP BY p HAVING current > 0 ORDER BY current DESC, p LIMIT 50",
		PatternExpr("Body"), listBodyLimit, where)
	return q + querySettings, append([]any{f.Start.UnixNano()}, args...)
}

func contextQuery(at time.Time, nodeName, fileName string, n int) (string, []any) {
	sel := "SELECT " + lineColumns(contextBodyLimit) + " FROM otel.logs WHERE ResourceAttributes['node_name'] = ? AND " +
		"LogAttributes['log.file.name'] = ? AND TimestampTime BETWEEN toDateTime(?) AND toDateTime(?) AND "
	q := "SELECT " + lineColumnNames + " FROM (" +
		"(" + sel + "Timestamp <= fromUnixTimestamp64Nano(?) ORDER BY Timestamp DESC LIMIT ?) UNION ALL " +
		"(" + sel + "Timestamp > fromUnixTimestamp64Nano(?) ORDER BY Timestamp LIMIT ?)" +
		") ORDER BY Timestamp" + querySettings
	return q, []any{
		nodeName, fileName, at.Add(-contextWindow).Unix(), at.Unix(), at.UnixNano(), n,
		nodeName, fileName, at.Unix(), at.Add(contextWindow).Unix(), at.UnixNano(), n,
	}
}

func facetsQueries(f Filter) (string, string, string, []any) {
	where, args := timeWhere(f.Start, f.End)
	facet := func(column string) string {
		return "SELECT " + column + " AS v, count() FROM otel.logs WHERE " + where +
			" GROUP BY v HAVING v != '' ORDER BY count() DESC LIMIT 200" + querySettings
	}
	return facet("ResourceAttributes['node_name']"), facet("ServiceName"), facet("LogAttributes['log.preset']"), args
}
