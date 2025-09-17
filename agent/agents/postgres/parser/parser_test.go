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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expectedResult struct {
	Tables []string `json:"tables"`
	Err    string   `json:"error"`
}

func TestExtractTables(t *testing.T) {
	files, err := filepath.Glob(filepath.FromSlash("./testdata/*.sql"))
	require.NoError(t, err)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			d, err := os.ReadFile(file) //nolint:gosec
			require.NoError(t, err)
			query := string(d)

			goldenFile := strings.TrimSuffix(file, ".sql") + ".json"
			d, err = os.ReadFile(goldenFile) //nolint:gosec
			require.NoError(t, err)
			var expected expectedResult
			err = json.Unmarshal(d, &expected)
			require.NoError(t, err)

			for name, f := range map[string]func(string) ([]string, error){
				"ExtractTables": ExtractTables,
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					actual, err := f(query)
					assert.Equal(t, expected.Tables, actual)
					if expected.Err != "" {
						require.EqualError(t, err, expected.Err, "err = %+v", err)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})
	}
}

var actualB interface{}

func BenchmarkExtractTables(b *testing.B) {
	files, err := filepath.Glob(filepath.FromSlash("./testdata/*.sql"))
	require.NoError(b, err)

	for _, file := range files {
		goldenFile := strings.TrimSuffix(file, ".sql") + ".json"
		name := strings.TrimSuffix(filepath.Base(file), ".log")
		b.Run(name, func(b *testing.B) {
			d, err := os.ReadFile(file) //nolint:gosec
			require.NoError(b, err)
			query := string(d)

			d, err = os.ReadFile(goldenFile) //nolint:gosec
			require.NoError(b, err)
			var expected expectedResult
			err = json.Unmarshal(d, &expected)
			require.NoError(b, err)

			b.SetBytes(int64(len(query)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				actualB, err = ExtractTables(query)
			}

			b.StopTimer()

			assert.Equal(b, expected.Tables, actualB.([]string))
			if expected.Err != "" {
				require.EqualError(b, err, expected.Err, "err = %+v", err)
			} else {
				require.NoError(b, err)
			}
		})
	}
}
