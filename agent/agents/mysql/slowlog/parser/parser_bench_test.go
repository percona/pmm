// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/percona/go-mysql/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkParser(b *testing.B) {
	files, err := filepath.Glob(filepath.FromSlash("./testdata/*.log"))
	require.NoError(b, err)
	for _, name := range files {
		benchmarkFile(b, name)
	}
}

func benchmarkFile(b *testing.B, name string) {
	b.Helper()

	s, err := os.Stat(name)
	require.NoError(b, err)

	b.Run(name, func(b *testing.B) {
		b.SetBytes(s.Size())
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			b.StopTimer()

			r, err := NewSimpleFileReader(name)
			assert.NoError(b, err)
			p := NewSlowLogParser(r, log.Options{})

			b.StartTimer()

			go p.Run()
			for p.Parse() != nil { //nolint:revive
			}

			b.StopTimer()

			assert.Equal(b, io.EOF, p.Err())
			assert.NoError(b, r.Close())
		}
	})
}
