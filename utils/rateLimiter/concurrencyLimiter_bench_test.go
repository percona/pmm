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

package rateLimiter

import (
	"runtime"
	"testing"
)

func BenchmarkConcurrencyLimiter_TryAcquireRelease(b *testing.B) {
	limiter := NewConcurrencyLimiter(1)
	b.ReportAllocs()

	for b.Loop() {
		if !limiter.TryAcquire() {
			b.Fatal("expected acquire to succeed")
		}
		limiter.Release()
	}
}

func BenchmarkConcurrencyLimiter_TryAcquireWhenExhausted(b *testing.B) {
	limiter := NewConcurrencyLimiter(0)
	b.ReportAllocs()

	for b.Loop() {
		if limiter.TryAcquire() {
			b.Fatal("expected acquire to fail for exhausted limiter")
		}
	}
}

func BenchmarkConcurrencyLimiter_ParallelAcquireRelease(b *testing.B) {
	limiter := NewConcurrencyLimiter(int32(runtime.GOMAXPROCS(0))) //nolint:gosec
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if limiter.TryAcquire() {
				limiter.Release()
			}
		}
	})
}
