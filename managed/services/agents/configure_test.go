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

package agents

import (
	"os"
	"testing"

	"github.com/percona/pmm/managed/models"
)

// TestMain entry point for all tests execution. Used for tests global configuration.
func TestMain(m *testing.M) {
	// replace hash func with stub because of high cyclomatic complexity
	// of bcrypt's internals. This allows much faster -race test runs.
	models.HashPassword = func(password, _ string) (string, error) {
		return password, nil
	}

	code := m.Run()

	os.Exit(code)
}
