// pmm-agent
// Copyright 2019 Percona LLC
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
	"io/ioutil"
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
		goldenFile := strings.TrimSuffix(file, ".sql") + ".json"
		name := strings.TrimSuffix(filepath.Base(file), ".log")
		t.Run(name, func(t *testing.T) {
			b, err := ioutil.ReadFile(file) //nolint:gosec
			require.NoError(t, err)
			query := string(b)

			b, err = ioutil.ReadFile(goldenFile) //nolint:gosec
			require.NoError(t, err)
			var expected expectedResult
			err = json.Unmarshal(b, &expected)
			require.NoError(t, err)

			actual, err := ExtractTables(query)
			assert.Equal(t, expected.Tables, actual)
			if expected.Err != "" {
				require.EqualError(t, err, expected.Err, "err = %+v", err)
			} else {
				require.NoError(t, err, "err = %+v", err)
			}
		})
	}
}
