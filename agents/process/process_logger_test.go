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

package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessLogger(t *testing.T) {
	tests := []struct {
		testName       string
		writerLines    int
		writeArgs      []string
		expectedLatest []string
		expectedLen    int
		expectedCap    int
	}{
		{
			"simple one",
			4,
			[]string{
				"text\n",
			},
			[]string{"text"},
			0,
			0,
		},
		{
			"two line in one write",
			4,
			[]string{
				"text\nsecond line\n",
			},
			[]string{"text", "second line"},
			0,
			0,
		},
		{
			"three line in two writes",
			4,
			[]string{
				"text\nsecond ",
				"line\nthird row\n",
			},
			[]string{"text", "second line", "third row"},
			0,
			0,
		},
		{
			"log overflow",
			2,
			[]string{
				"text\nsecond ",
				"line\nthird row\n",
			},
			[]string{"second line", "third row"},
			0,
			0,
		},
		{
			"another log overflow",
			2,
			[]string{
				"text\nsecond ",
				"line\nthird row\n",
				"fourth ",
				"line\nlast row\n",
			},
			[]string{"fourth line", "last row"},
			0,
			0,
		},
		{
			"don't write not finished line",
			10,
			[]string{
				"text\nsecond line",
			},
			[]string{"text"},
			11,
			16,
		},
		{
			"preserve empty lines",
			10,
			[]string{
				"\n1\n\n2\n\n",
			},
			[]string{"", "1", "", "2", ""},
			0,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			pl := newProcessLogger(nil, tt.writerLines)
			for _, arg := range tt.writeArgs {
				_, err := pl.Write([]byte(arg))
				require.NoError(t, err)
			}
			latest := pl.Latest()
			assert.Equal(t, tt.expectedLatest, latest)
			assert.Len(t, pl.buf, tt.expectedLen, "Unexpected buf len.")
			assert.Equal(t, tt.expectedCap, cap(pl.buf), "Unexpected buf cap. buf: %s", pl.buf)
		})
	}
}
