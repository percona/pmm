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

package models

import "time"

// Sparklines are capped at optimalAmountOfPoint points, so that a wide time range does not
// return a huge response. Below minFullTimeFrame that cap does not bite and the series has
// one point per minute instead.
const (
	optimalAmountOfPoint = 120
	minFullTimeFrame     = 2 * time.Hour
	secondsPerMinute     = 60
)

// sparklineLayout describes how a requested period is divided into sparkline points.
type sparklineLayout struct {
	// periodStartFromSec and periodStartToSec are the requested bounds aligned down to a
	// whole minute. Callers query ClickHouse with these rather than with the raw request
	// bounds, so that points line up with the one-minute rows in the metrics table, and
	// they place the points on the same grid when filling gaps in the series.
	periodStartFromSec int64
	periodStartToSec   int64
	// amountOfPoints is how many points the series has, and timeFrame is how many seconds
	// each one covers.
	amountOfPoints int64
	timeFrame      int64
}

// newSparklineLayout works out how the given period is divided into sparkline points. It
// aligns both bounds down to a whole minute itself and returns them, so that no caller has
// to repeat that arithmetic and no caller can reach the division below with bounds that
// are not aligned.
//
// A period of less than a minute -- both bounds inside the same minute, or a reversed
// range -- leaves no minutes to spread across points, which would make every divisor
// below zero. Such a period is reported as a single one-minute point instead.
func newSparklineLayout(periodStartFromSec, periodStartToSec int64) sparklineLayout {
	from := periodStartFromSec / secondsPerMinute * secondsPerMinute
	to := periodStartToSec / secondsPerMinute * secondsPerMinute

	timePeriod := max(to-from, secondsPerMinute)

	// If time range is bigger then two hour - amount of sparklines points = 120 to avoid huge data in response.
	// Otherwise amount of sparklines points is equal to minutes in time range to not mess up calculation.
	amountOfPoints := int64(optimalAmountOfPoint)
	// reduce amount of point if period less then 2h.
	if timePeriod < int64(minFullTimeFrame.Seconds()) {
		// minimum point is 1 minute
		amountOfPoints = timePeriod / secondsPerMinute
	}

	// how many full minutes we can fit into given amount of points.
	minutesInPoint := timePeriod / secondsPerMinute / amountOfPoints
	// we need aditional point to show this minutes
	remainder := (timePeriod / secondsPerMinute) % amountOfPoints
	amountOfPoints += remainder / minutesInPoint

	return sparklineLayout{
		periodStartFromSec: from,
		periodStartToSec:   to,
		amountOfPoints:     amountOfPoints,
		timeFrame:          minutesInPoint * secondsPerMinute,
	}
}

// PeriodDuration is the length of a requested period in seconds, floored at one second.
//
// A report may be requested for a single instant, where the per-second rates derived from
// this value -- in Go, and in the SQL templates that divide by it -- would otherwise
// divide by zero and yield +Inf, which the generated API clients cannot decode into their
// float fields. One second is the smallest floor that leaves every real duration
// untouched, so this is a no-op for every period of a second or more.
//
// Note that a sparkline over the same sub-minute period reports it as one whole minute
// (see newSparklineLayout), so the rates derived from this value disagree with the
// sparkline there. Flooring at a minute instead would change the rates of sub-minute
// periods that work today, so the disagreement is left in place.
func PeriodDuration(periodStartFromSec, periodStartToSec int64) int64 {
	return max(periodStartToSec-periodStartFromSec, 1)
}
