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

package ansible

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnsible(t *testing.T) {
	playbook, err := filepath.Abs(filepath.Join("..", "..", "ansible", "playbook", "tasks", "update.yml"))
	require.NoError(t, err)

	t.Run("SyntaxCheck", func(t *testing.T) {
		err := RunPlaybook(context.Background(), playbook, &RunPlaybookOpts{
			ExtraFlags: []string{"--syntax-check"},
		},
		)
		require.NoError(t, err)
	})

	// playbooks are tested by `make check`
}
