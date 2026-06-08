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

package ddsketch

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

const tol = Alpha + 1e-9

// exactQuantile mirrors Quantile's rank rule on a sorted slice (each value its own bucket).
func exactQuantile(sorted []float64, q float64) float64 {
	n := len(sorted)
	rank := q * float64(n-1)
	p := 0
	for p < n-1 && float64(p+1) <= rank {
		p++
	}
	return sorted[p]
}

func TestLen(t *testing.T) {
	t.Parallel()
	// Layout is deterministic and large enough to cover the range with overflow buckets.
	require.Equal(t, (iMax-iMin+1)+2, Len())
	require.Len(t, New(), Len())
	require.Greater(t, Len(), 400)
}

func TestRelativeErrorBound(t *testing.T) {
	t.Parallel()
	for v := MinValue * 2; v < MaxValue/2; v *= 1.1 {
		counts := New()
		Add(counts, v)
		got := Quantile(counts, 0.5)
		relErr := math.Abs(got-v) / v
		require.LessOrEqualf(t, relErr, tol, "v=%g got=%g relErr=%g", v, got, relErr)
	}
}

func TestQuantileAccuracy(t *testing.T) {
	t.Parallel()
	// Deterministic ramp: 1ms .. 10000ms.
	sorted := make([]float64, 0, 10000)
	counts := New()
	for i := 1; i <= 10000; i++ {
		v := float64(i) / 1000.0
		sorted = append(sorted, v)
		Add(counts, v)
	}
	for _, q := range []float64{0, 0.5, 0.9, 0.99, 1} {
		got := Quantile(counts, q)
		want := exactQuantile(sorted, q)
		relErr := math.Abs(got-want) / want
		require.LessOrEqualf(t, relErr, tol, "q=%g got=%g want=%g relErr=%g", q, got, want, relErr)
	}
}

func TestMergeMatchesUnion(t *testing.T) {
	t.Parallel()
	a, b := New(), New()
	union := New()
	for i := 1; i <= 500; i++ {
		va := float64(i) / 1000.0
		vb := float64(i) / 100.0
		Add(a, va)
		Add(b, vb)
		Add(union, va)
		Add(union, vb)
	}

	merged := New()
	Merge(merged, a)
	Merge(merged, b)
	require.Equal(t, union, merged) // identical sketches => identical quantiles
}

func TestUnderflowOverflow(t *testing.T) {
	t.Parallel()
	counts := New()
	Add(counts, MinValue/10) // underflow
	Add(counts, MaxValue*10) // overflow
	require.Equal(t, uint64(1), counts[0])
	require.Equal(t, uint64(1), counts[Len()-1])
	require.InDelta(t, MinValue, Quantile(counts, 0), 1e-9)
	require.InDelta(t, MaxValue, Quantile(counts, 1), 1e-9)
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	require.Zero(t, Quantile(New(), 0.5))
}

func TestToWire(t *testing.T) {
	t.Parallel()
	require.Nil(t, ToWire(New()), "empty sketch -> nil wire map")

	dense := New()
	for i := 1; i <= 100; i++ {
		Add(dense, float64(i)/1000.0)
	}
	wire := ToWire(dense)
	require.NotEmpty(t, wire)

	// Densify the wire map back and confirm the quantile survives the round-trip.
	back := New()
	for idx, c := range wire {
		back[idx] = c
	}
	require.InDelta(t, Quantile(dense, 0.99), Quantile(back, 0.99), 1e-9)
}
