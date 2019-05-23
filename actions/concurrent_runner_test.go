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
)

func TestConcurrentRunnerRun(t *testing.T) {
	t.Parallel()

	cr := NewConcurrentRunner(context.Background(), 0)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "echo", []string{"test"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "echo", []string{"test2"})

	cr.Start(a1)
	cr.Start(a2)

	expected := []string{"test\n", "test2\n"}
	for i := 0; i < 2; i++ {
		a := <-cr.Results()
		assert.Contains(t, expected, string(a.Output))
	}
}

func TestConcurrentRunnerTimeout(t *testing.T) {
	t.Parallel()

	cr := NewConcurrentRunner(context.Background(), time.Second)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "sleep", []string{"30"})

	cr.Start(a1)
	cr.Start(a2)

	// check Action returns proper errors and output.
	expected := []string{"signal: killed", "signal: killed"}
	expectedOut := []string{"", ""}
	for i := 0; i < 2; i++ {
		r := <-cr.Results()
		assert.Contains(t, expected, r.Error.Error())
		assert.Contains(t, expectedOut, string(r.Output))
	}

	assert.NotContains(t, cr.actionsCancel, a1.ID())
	assert.NotContains(t, cr.actionsCancel, a2.ID())
}

func TestConcurrentRunnerStop(t *testing.T) {
	t.Parallel()

	cr := NewConcurrentRunner(context.Background(), 0)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "sleep", []string{"30"})

	cr.Start(a1)
	cr.Start(a2)

	<-time.After(time.Second)

	cr.Stop(a1.ID())
	cr.Stop(a2.ID())

	// check Action returns proper errors and output.
	expected := []string{"signal: killed", "signal: killed"}
	expectedOut := []string{"", ""}
	for i := 0; i < 2; i++ {
		r := <-cr.Results()
		assert.Contains(t, expected, r.Error.Error())
		assert.Contains(t, expectedOut, string(r.Output))
	}

	assert.NotContains(t, cr.actionsCancel, a1.ID())
	assert.NotContains(t, cr.actionsCancel, a2.ID())
}

func TestConcurrentRunnerCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cr := NewConcurrentRunner(ctx, 0)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "sleep", []string{"30"})

	cr.Start(a1)
	cr.Start(a2)

	cancel()

	expected := []string{"context canceled", "context canceled"}
	expectedOut := []string{"", ""}
	for i := 0; i < 2; i++ {
		r := <-cr.Results()
		assert.Contains(t, expected, r.Error.Error())
		assert.Contains(t, expectedOut, string(r.Output))
	}

	assert.NotContains(t, cr.actionsCancel, a1.ID())
	assert.NotContains(t, cr.actionsCancel, a2.ID())
}

func TestConcurrentRunnerCancelEmpty(t *testing.T) {
	t.Skip("https://jira.percona.com/browse/PMM-4112")
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cr := NewConcurrentRunner(ctx, 0)
	a := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})

	go cancel()
	cr.Start(a)

	expected := []string{"context canceled", "context canceled"}
	expectedOut := []string{"", ""}
	r := <-cr.Results()
	assert.Contains(t, expected, r.Error.Error())
	assert.Contains(t, expectedOut, string(r.Output))

	assert.NotContains(t, cr.actionsCancel, a.ID())
}
