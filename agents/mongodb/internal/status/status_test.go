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
