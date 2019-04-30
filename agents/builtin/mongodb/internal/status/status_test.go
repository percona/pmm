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

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStats struct {
	FieldOne   int64  `name:"field-one"`
	FieldTwo   int64  `name:"field-two"`
	FieldThree string `name:"field-three"`
}

func TestStatus(t *testing.T) {
	// we need at least one test per package to correctly calculate coverage
	s := &testStats{1, 2, "test"}
	testState := New(s)

	expected := map[string]string{
		"field-one":   "1",
		"field-two":   "2",
		"field-three": "test",
	}

	assert.Equal(t, expected, testState.Map())
}
