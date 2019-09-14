// pmm-agent
// Copyright 2019 Percona LLC
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
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// assertResults checks expected results in any order.
func assertResults(t *testing.T, cr *ConcurrentRunner, expected ...ActionResult) {
	t.Helper()

	actual := make([]ActionResult, len(expected))
	for i := range expected {
		r := <-cr.Results()
		if len(r.Output) == 0 {
			r.Output = nil
		}
		actual[i] = r
	}

	sort.Slice(expected, func(i, j int) bool { return expected[i].ID < expected[j].ID })
	sort.Slice(actual, func(i, j int) bool { return actual[i].ID < actual[j].ID })
	assert.Equal(t, expected, actual)
}

func TestConcurrentRunnerRun(t *testing.T) {
	t.Parallel()

	cr := NewConcurrentRunner(context.Background(), 5*time.Second)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "echo", []string{"test"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "echo", []string{"test2"})

	cr.Start(a1)
	cr.Start(a2)

	expected := []ActionResult{
		{ID: "/action_id/6a479303-5081-46d0-baa0-87d6248c987b", Output: []byte("test\n")},
		{ID: "/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", Output: []byte("test2\n")},
	}
	assertResults(t, cr, expected...)
	assert.Empty(t, cr.actionsCancel)
}

func TestConcurrentRunnerTimeout(t *testing.T) {
	t.Parallel()

	cr := NewConcurrentRunner(context.Background(), time.Second)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "sleep", []string{"30"})

	cr.Start(a1)
	cr.Start(a2)

	// https://github.com/golang/go/issues/21880
	expected := []ActionResult{
		{ID: "/action_id/6a479303-5081-46d0-baa0-87d6248c987b", Error: "signal: killed"},
		{ID: "/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", Error: "signal: killed"},
	}
	assertResults(t, cr, expected...)
	assert.Empty(t, cr.actionsCancel)
}

func TestConcurrentRunnerStop(t *testing.T) {
	t.Parallel()

	cr := NewConcurrentRunner(context.Background(), 5*time.Second)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "sleep", []string{"30"})

	cr.Start(a1)
	cr.Start(a2)

	<-time.After(time.Second)

	cr.Stop(a1.ID())
	cr.Stop(a2.ID())

	// https://github.com/golang/go/issues/21880
	expected := []ActionResult{
		{ID: "/action_id/6a479303-5081-46d0-baa0-87d6248c987b", Error: "signal: killed"},
		{ID: "/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", Error: "signal: killed"},
	}
	assertResults(t, cr, expected...)
	assert.Empty(t, cr.actionsCancel)
}

func TestConcurrentRunnerCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cr := NewConcurrentRunner(ctx, 5*time.Second)
	a1 := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})
	a2 := NewProcessAction("/action_id/84140ab2-612d-4d93-9360-162a4bd5de14", "sleep", []string{"30"})

	cr.Start(a1)
	cr.Start(a2)

	cancel()

	// Unlike other tests, there we mostly see "context canceled", but "signal: killed" still happens.
	// Check both.
	expected := make([]ActionResult, 2)
	expected[0] = <-cr.Results()
	expected[1] = <-cr.Results()
	sort.Slice(expected, func(i, j int) bool { return expected[i].ID < expected[j].ID })
	assert.Equal(t, expected[0].ID, "/action_id/6a479303-5081-46d0-baa0-87d6248c987b")
	assert.Contains(t, []string{"signal: killed", context.Canceled.Error()}, expected[0].Error)
	assert.Equal(t, expected[1].ID, "/action_id/84140ab2-612d-4d93-9360-162a4bd5de14")
	assert.Contains(t, []string{"signal: killed", context.Canceled.Error()}, expected[0].Error)
	assert.Empty(t, cr.actionsCancel)
}

func TestConcurrentRunnerCancelEmpty(t *testing.T) {
	t.Skip("https://jira.percona.com/browse/PMM-4112")
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cr := NewConcurrentRunner(ctx, 5*time.Second)
	a := NewProcessAction("/action_id/6a479303-5081-46d0-baa0-87d6248c987b", "sleep", []string{"20"})

	go cancel()
	cr.Start(a)

	expected := []ActionResult{
		{ID: "/action_id/6a479303-5081-46d0-baa0-87d6248c987b", Error: context.Canceled.Error()},
	}
	assertResults(t, cr, expected...)
	assert.Empty(t, cr.actionsCancel)
}
