// Copyright (C) 2024 Percona LLC
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

package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackages(t *testing.T) {
	cmd := exec.Command("pmm-update", "-h")
	b, err := cmd.CombinedOutput()
	if err != nil {
		// This branch is required for tests with pmm-server:2.0.0
		// In this case the exit code is 2.
		e, ok := err.(*exec.ExitError) //nolint:errorlint
		require.True(t, ok)

		sb := string(b)
		assert.Equal(t, 2, e.ExitCode())
		assert.True(t, strings.Contains(sb, "ProjectName: pmm-update"), sb)
	} else {
		assert.NoError(t, err, string(b))
	}

	out := string(b)
	assert.False(t, strings.Contains(out, "-httptest.serve"), `pmm-update should not import package "net/http/httptest"`)
	assert.False(t, strings.Contains(out, "-test.run"), `pmm-update should not import package "testing"`)
}
