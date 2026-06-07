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

// Package ddsketch implements a DDSketch-style log-bucket histogram for latency
// percentiles. Values map to a fixed, frozen bucket layout; bucket counts are
// stored as a plain []uint64, merged by element-wise addition, and queried at
// read time. The layout is a compile-time constant — not configurable — so every
// stored array is mergeable across service restarts and ClickHouse upgrades.
package ddsketch

import "math"

const (
	// Alpha is the frozen relative accuracy: every returned quantile is within
	// Alpha of the true value. Changing it is an explicit schema migration.
	Alpha = 0.02
	// MinValue and MaxValue bound the represented latency range, in seconds.
	MinValue = 1e-6
	MaxValue = 1000.0
	// LayoutVersion tags stored arrays so a future layout change is detectable.
	LayoutVersion = 1
)

var (
	gamma    = (1 + Alpha) / (1 - Alpha)
	logGamma = math.Log(gamma)
	iMin     = int(math.Ceil(math.Log(MinValue) / logGamma))
	iMax     = int(math.Ceil(math.Log(MaxValue) / logGamma))
	// size = in-range buckets + underflow (index 0) + overflow (last index).
	size = (iMax - iMin + 1) + 2
)

// Len returns the fixed sketch array length. Storage and ingestion must agree on it.
func Len() int { return size }

// New returns a zeroed sketch array.
func New() []uint64 { return make([]uint64, size) }

// index returns the array position for value v.
func index(v float64) int {
	switch {
	case v <= MinValue:
		return 0
	case v > MaxValue:
		return size - 1
	default:
		return int(math.Ceil(math.Log(v)/logGamma)) - iMin + 1
	}
}

// value returns the representative latency for array position a.
func value(a int) float64 {
	switch {
	case a <= 0:
		return MinValue
	case a >= size-1:
		return MaxValue
	default:
		i := a - 1 + iMin
		return 2 * math.Pow(gamma, float64(i)) / (gamma + 1)
	}
}

// BucketBounds returns the latency range (lo, hi] that array position a covers.
func BucketBounds(a int) (lo, hi float64) {
	switch {
	case a <= 0:
		return 0, MinValue
	case a >= size-1:
		return MaxValue, math.Inf(1)
	default:
		i := a - 1 + iMin
		return math.Pow(gamma, float64(i-1)), math.Pow(gamma, float64(i))
	}
}

// ToWire converts a dense sketch into the wire bucket map (index -> count),
// returning nil when there is no data so the field stays empty.
func ToWire(dense []uint64) map[uint32]uint64 {
	out := make(map[uint32]uint64)
	for i, c := range dense {
		if c > 0 {
			out[uint32(i)] = c
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Add records one observation of v into counts.
func Add(counts []uint64, v float64) { counts[index(v)]++ }

// AddN records n observations of v into counts.
func AddN(counts []uint64, v float64, n uint64) { counts[index(v)] += n }

// Merge adds src into dst element-wise. Both must share the frozen layout.
func Merge(dst, src []uint64) {
	for i := range dst {
		dst[i] += src[i]
	}
}

// QuantileFromMap returns the q-th quantile from a sparse bucket map (as stored in
// ClickHouse), densifying it into the fixed layout first.
func QuantileFromMap(counts map[uint16]uint64, q float64) float64 {
	dense := New()
	for idx, c := range counts {
		if int(idx) < len(dense) {
			dense[idx] += c
		}
	}
	return Quantile(dense, q)
}

// Quantile returns the q-th quantile (0..1) of the distribution in counts.
func Quantile(counts []uint64, q float64) float64 {
	var total uint64
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0
	}
	rank := q * float64(total-1)
	var cum uint64
	for a, c := range counts {
		cum += c
		if float64(cum) > rank {
			return value(a)
		}
	}
	return value(size - 1)
}
