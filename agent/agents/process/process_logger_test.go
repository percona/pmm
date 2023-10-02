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
		redactWords    []string
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			nil,
			[]string{"", "1", "", "2", ""},
			0,
			0,
		},
		{
			"redact keywords",
			3,
			[]string{
				"text\nsecond ",
				"line\nthird row line\n",
				"fourth ",
				"line\nlast row\n",
			},
			[]string{"row"},
			[]string{"third *** line", "fourth line", "last ***"},
			0,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			pl := newProcessLogger(nil, tt.writerLines, tt.redactWords)
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
