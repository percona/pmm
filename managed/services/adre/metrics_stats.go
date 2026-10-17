// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"math"
	"sort"
	"time"
)

const (
	defaultZScoreThreshold = 2.0
	defaultMaxAnomalies    = 10
)

// SeriesStats holds percentile summary for a timeseries window.
type SeriesStats struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P25    float64 `json:"p25"`
	P75    float64 `json:"p75"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

// ChangePoint marks the largest step shift in a window.
type ChangePoint struct {
	Bucket   string  `json:"bucket"`
	Value    float64 `json:"value"`
	Delta    float64 `json:"delta"`
	DeltaPct float64 `json:"delta_pct"`
}

// Anomaly marks a bucket whose z-score exceeds the threshold.
type Anomaly struct {
	Bucket string  `json:"bucket"`
	Value  float64 `json:"value"`
	ZScore float64 `json:"z_score"`
}

func computeSeriesStats(values []float64) SeriesStats {
	if len(values) == 0 {
		return SeriesStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	return SeriesStats{
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Mean:   mean,
		Median: percentileSorted(sorted, 0.50), //nolint:mnd
		P25:    percentileSorted(sorted, 0.25), //nolint:mnd
		P75:    percentileSorted(sorted, 0.75), //nolint:mnd
		P95:    percentileSorted(sorted, 0.95), //nolint:mnd
		P99:    percentileSorted(sorted, 0.99), //nolint:mnd
	}
}

func percentileSorted(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	weight := pos - float64(lo)
	return sorted[lo]*(1-weight) + sorted[hi]*weight
}

func findChangePoints(timestamps []time.Time, values []float64, topN int) []ChangePoint {
	if len(values) < 2 || topN <= 0 {
		return nil
	}
	type candidate struct {
		idx      int
		delta    float64
		deltaPct float64
	}
	candidates := make([]candidate, 0, len(values)-1)
	for i := 1; i < len(values); i++ {
		prev := values[i-1]
		cur := values[i]
		delta := cur - prev
		deltaPct := 0.0
		if prev != 0 {
			deltaPct = (delta / math.Abs(prev)) * 100 //nolint:mnd
		} else if cur != 0 {
			deltaPct = 100
		}
		candidates = append(candidates, candidate{idx: i, delta: delta, deltaPct: deltaPct})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return math.Abs(candidates[i].deltaPct) > math.Abs(candidates[j].deltaPct)
	})
	if topN > len(candidates) {
		topN = len(candidates)
	}
	out := make([]ChangePoint, 0, topN)
	for i := range topN {
		c := candidates[i]
		out = append(out, ChangePoint{
			Bucket:   timestamps[c.idx].UTC().Format(time.RFC3339),
			Value:    values[c.idx],
			Delta:    c.delta,
			DeltaPct: c.deltaPct,
		})
	}
	return out
}

func findAnomalies(timestamps []time.Time, values []float64, threshold float64) []Anomaly {
	return findAnomaliesTopN(timestamps, values, threshold, defaultMaxAnomalies)
}

func findAnomaliesTopN(timestamps []time.Time, values []float64, threshold float64, topN int) []Anomaly {
	if len(values) < 2 || topN <= 0 {
		return nil
	}
	stats := computeSeriesStats(values)
	var variance float64
	for _, v := range values {
		d := v - stats.Mean
		variance += d * d
	}
	variance /= float64(len(values))
	stddev := math.Sqrt(variance)
	if stddev == 0 {
		return nil
	}
	type candidate struct {
		idx int
		z   float64
	}
	candidates := make([]candidate, 0)
	for i, v := range values {
		z := (v - stats.Mean) / stddev
		if math.Abs(z) <= threshold {
			continue
		}
		candidates = append(candidates, candidate{idx: i, z: z})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return math.Abs(candidates[i].z) > math.Abs(candidates[j].z)
	})
	if topN > len(candidates) {
		topN = len(candidates)
	}
	out := make([]Anomaly, 0, topN)
	for i := range topN {
		c := candidates[i]
		out = append(out, Anomaly{
			Bucket: timestamps[c.idx].UTC().Format(time.RFC3339),
			Value:  values[c.idx],
			ZScore: c.z,
		})
	}
	return out
}
