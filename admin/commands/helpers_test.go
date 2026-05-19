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

package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDurationString(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		require.Empty(t, DurationString(nil))
	})

	for _, testCase := range []struct {
		name     string
		value    time.Duration
		expected string
	}{
		{
			name:     "sub-second",
			value:    500 * time.Millisecond,
			expected: "0.5s",
		},
		{
			name:     "multi-minute",
			value:    90 * time.Second,
			expected: "90s",
		},
		{
			name:     "hour",
			value:    time.Hour,
			expected: "3600s",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.expected, DurationString(&testCase.value))
		})
	}
}
