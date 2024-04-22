// Copyright (C) 2024 Percona LLC
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
	timeout := time.Second
	p := NewProcessAction(id, timeout, cmd, nil)

	// run
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := p.Run(ctx)

	// check
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Equal(t, id, p.ID())
	assert.Equal(t, timeout, p.Timeout())
	assert.Equal(t, cmd, p.Type())
}

func TestProcessActionRunAndCancel(t *testing.T) {
	t.Parallel()

	// setup
	p := NewProcessAction("/action_id/14b2422d-32ec-44fb-9019-8b70e3cc8a3a", time.Second, "sleep", []string{"10"})

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout())
	// run
	time.AfterFunc(time.Millisecond, cancel)
	_, err := p.Run(ctx)

	// check
	assert.Error(t, err)
}
