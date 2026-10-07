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

package mcp

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// day is the unit of the "d" suffix in relative time expressions.
const day = 24 * time.Hour

// maxLookbackYears bounds how far back a time expression may reach, relative
// ("now-2d") or absolute (RFC3339). It is far beyond any PMM retention window,
// keeps the relative multiplication inside int64, and stops a caller from
// sending QAN a scan window measured in millennia.
const (
	maxLookbackYears = 10
	maxLookback      = maxLookbackYears * 365 * day
)

// maxLookahead bounds how far into the future an RFC3339 time may reach; a day
// covers any clock skew.
const maxLookahead = day

// relativeTime matches "now-1h", "now - 30m", "now-2d", "now-45s".
var relativeTime = regexp.MustCompile(`(?i)^now\s*-\s*(\d+)\s*([smhd])$`)

// parseTime resolves "now", a relative expression such as "now-1h", or an
// RFC3339 timestamp. PMM's QAN endpoints accept absolute timestamps only, so
// relative expressions are converted here.
func parseTime(expr string, now time.Time) (time.Time, error) {
	t := strings.TrimSpace(expr)
	if t == "" || strings.EqualFold(t, "now") {
		return now, nil
	}

	if m := relativeTime.FindStringSubmatch(t); m != nil {
		unit := map[string]time.Duration{"s": time.Second, "m": time.Minute, "h": time.Hour, "d": day}[strings.ToLower(m[2])]
		// Bound n before multiplying: n*unit wraps int64 nanoseconds for large n, and
		// a number too long for int64 is out of range too, not malformed.
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || n > int64(maxLookback/unit) {
			return time.Time{}, outOfRange(expr)
		}
		return now.Add(-time.Duration(n) * unit), nil
	}

	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return time.Time{}, newToolError(codeInvalidInput,
			"invalid time '%s': use RFC3339 (2026-09-14T10:00:00Z) or a relative expression such as now-1h", expr)
	}
	if parsed.Before(now.Add(-maxLookback)) || parsed.After(now.Add(maxLookahead)) {
		return time.Time{}, outOfRange(expr)
	}
	return parsed, nil
}

func outOfRange(expr string) *toolError {
	return newToolError(codeInvalidInput, "time '%s' is out of range; the maximum lookback is %d years, and times may be at most %dh ahead",
		expr, maxLookbackYears, int(maxLookahead/time.Hour))
}

// parseWindow resolves a period_from / period_to pair with defaults now-1h / now.
func parseWindow(from, to string, now time.Time) (time.Time, time.Time, error) {
	if from == "" {
		from = "now-1h"
	}
	start, err := parseTime(from, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseTime(to, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, newToolError(codeInvalidInput, "period_to (%s) must be after period_from (%s)", to, from)
	}
	return start, end, nil
}

// fmtNum renders a metric with at most three decimals, like the QAN UI.
func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// fmtNum3 rounds to three decimals before rendering.
func fmtNum3(f float64) string {
	return fmtNum(float64(int64(f*1000+0.5)) / 1000) //nolint:mnd
}

func fmtSeconds(f float64) string {
	return fmtNum3(f) + "s"
}
