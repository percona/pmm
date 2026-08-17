// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	limiter := NewConcurrencyLimiter(int32(runtime.GOMAXPROCS(0))) //nolint:nolintlint
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if limiter.TryAcquire() {
				limiter.Release()
			}
		}
	})
}
