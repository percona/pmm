// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package investigations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRemediationSteps_skipsCodeFences(t *testing.T) {
	t.Parallel()
	// Holmes wraps commands in fenced blocks; splitting line-by-line must not turn the ``` markers into
	// steps (which the UI would render as empty code boxes).
	content := "1. Verify ownership.\n" +
		"```bash\n" +
		"ls -ld /var/lib/mysql\n" +
		"chown -R mysql:mysql /var/lib/mysql\n" +
		"```\n" +
		"2. Restart MySQL.\n" +
		"```\n" +
		"systemctl restart mysql\n" +
		"```"
	steps := parseRemediationSteps(content)
	assert.Equal(t, []string{
		"Verify ownership.",
		"ls -ld /var/lib/mysql",
		"chown -R mysql:mysql /var/lib/mysql",
		"Restart MySQL.",
		"systemctl restart mysql",
	}, steps)
	for _, s := range steps {
		assert.False(t, strings.HasPrefix(s, "```"), "no step should be a code-fence marker: %q", s)
	}
}
