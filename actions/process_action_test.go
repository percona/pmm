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

package actions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessActionRun(t *testing.T) {
	t.Parallel()

	// setup
	id := "/action_id/6a479303-5081-46d0-baa0-87d6248c987b"
	cmd := "echo"
	p := NewProcessAction(id, cmd, nil)

	// run
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	got, err := p.Run(ctx)

	// check
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Equal(t, id, p.ID())
	assert.Equal(t, cmd, p.Type())
}

func TestProcessActionRunAndCancel(t *testing.T) {
	t.Parallel()

	// setup
	p := NewProcessAction("/action_id/14b2422d-32ec-44fb-9019-8b70e3cc8a3a", "sleep", []string{"10"})

	ctx, cancel := context.WithCancel(context.Background())
	// run
	time.AfterFunc(time.Millisecond, cancel)
	_, err := p.Run(ctx)

	// check
	assert.Error(t, err)
}
