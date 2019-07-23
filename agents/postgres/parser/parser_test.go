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
