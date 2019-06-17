// pmm-agent
// Copyright (C) 2018 Percona LLC
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
			for p.Parse() != nil {
			}

			b.StopTimer()

			assert.Equal(b, io.EOF, p.Err())
			assert.NoError(b, r.Close())
		}
	})
}
