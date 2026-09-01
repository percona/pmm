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

package tests

import (
	"os"
	"testing"
)

// UnsetEnv removes an environment variable for the duration of the test, restoring it afterwards.
//
// There is no counterpart to t.Setenv for unsetting, and setting a variable to an empty string is
// not the same thing: envvars.ParseEnvVars rejects an empty boolean as a configuration error, so a
// test that needs the "not set at all" case has to actually unset it.
func UnsetEnv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if !ok {
		return
	}

	// Registers the cleanup that puts the original value back, then drops it for this test.
	t.Setenv(key, value)
	err := os.Unsetenv(key)
	if err != nil {
		t.Fatalf("failed to unset %s: %s", key, err)
	}
}
