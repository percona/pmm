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

package truncate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	for q, expected := range map[string]struct {
		query     string
		truncated bool
	}{
		"abcd":    {"abcd", false},
		"абвг":    {"абвг", true}, // 8 runes due to Unicode
		"abcde":   {"abcde", false},
		"абвгд":   {"абвгд", true}, // 10 runes due to Unicode
		"абвгде":  {"а ...", true},
		"абвгдеё": {"а ...", true},

		// Unicode replacement characters
		"\xff\xff\xff\xff\xff":     {"\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD", false},
		"\xff\xff\xff\xff\xff\xff": {"\uFFFD ...", true},
	} {
		query, truncated := Query(q, 5, GetDefaultMaxQueryLength())
		assert.Equal(t, expected.query, query)
		assert.Equal(t, expected.truncated, truncated)
	}
}
