// Copyright (C) 2023 Percona LLC
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

package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected bool
	}{
		{
			name:     "not set",
			envKey:   "TEST_BOOL_NOT_SET",
			expected: false,
		},
		{
			name:     "true",
			envKey:   "TEST_BOOL_TRUE",
			envValue: "true",
			expected: true,
		},
		{
			name:     "false",
			envKey:   "TEST_BOOL_FALSE",
			envValue: "false",
			expected: false,
		},
		{
			name:     "invalid",
			envKey:   "TEST_BOOL_INVALID",
			envValue: "invalid",
			expected: false,
		},
		{
			name:     "1",
			envKey:   "TEST_BOOL_1",
			envValue: "1",
			expected: true,
		},
		{
			name:     "0",
			envKey:   "TEST_BOOL_0",
			envValue: "0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}
			result := GetBool(tt.envKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSEPEnabled(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		envValue string
		expected bool
	}{
		{
			name:     "not set",
			expected: false,
		},
		{
			name:     "empty",
			set:      true,
			envValue: "",
			expected: false,
		},
		{
			name:     "1",
			set:      true,
			envValue: "1",
			expected: true,
		},
		{
			name:     "true",
			set:      true,
			envValue: "true",
			expected: true,
		},
		{
			name:     "0",
			set:      true,
			envValue: "0",
			expected: false,
		},
		{
			name:     "false",
			set:      true,
			envValue: "false",
			expected: false,
		},
		// The container entrypoint treats only "1" and "true" as enabled, so
		// these must not enable SEP here either: the API would advertise
		// navigation that nginx has no location block to route.
		{
			name:     "True",
			set:      true,
			envValue: "True",
			expected: false,
		},
		{
			name:     "yes",
			set:      true,
			envValue: "yes",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv(EnableSEP, tt.envValue)
			} else {
				// Setenv first, so the original value is restored on cleanup.
				t.Setenv(EnableSEP, "")
				os.Unsetenv(EnableSEP)
			}

			assert.Equal(t, tt.expected, SEPEnabled())
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected []string
	}{
		{
			name:     "not set",
			envKey:   "TEST_SLICE_NOT_SET",
			expected: []string{},
		},
		{
			name:     "empty",
			envKey:   "TEST_SLICE_EMPTY",
			envValue: "",
			expected: []string{},
		},
		{
			name:     "single",
			envKey:   "TEST_SLICE_SINGLE",
			envValue: "a",
			expected: []string{"a"},
		},
		{
			name:     "multiple",
			envKey:   "TEST_SLICE_MULTIPLE",
			envValue: "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with empty",
			envKey:   "TEST_SLICE_EMPTY",
			envValue: "a,,c",
			expected: []string{"a", "", "c"},
		},
		{
			name:     "leading comma",
			envKey:   "TEST_SLICE_LEADING",
			envValue: ",a",
			expected: []string{"", "a"},
		},
		{
			name:     "trailing comma",
			envKey:   "TEST_SLICE_TRAILING",
			envValue: "a,",
			expected: []string{"a", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}
			result := GetStringSlice(tt.envKey)
			assert.Len(t, tt.expected, len(result))
			for i, v := range result {
				assert.Equal(t, tt.expected[i], v)
			}
		})
	}
}
