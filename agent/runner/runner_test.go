// Copyright (C) 2023 Percona LLC
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

package runner

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/runner/actions"
	"github.com/percona/pmm/agent/runner/jobs"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// assertActionResults checks expected results in any order.
func assertActionResults(t *testing.T, cr *Runner, expected ...*agentv1.ActionResultRequest) {
	t.Helper()

	actual := make([]agentv1.AgentRequestPayload, len(expected))
	for i := range expected {
		actual[i] = <-cr.ActionsResults()
	}
	assert.ElementsMatch(t, expected, actual)
}

func TestConcurrentRunnerRun(t *testing.T) {
	t.Parallel()
	cr := New(0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)
	a1 := actions.NewProcessAction("6a479303-5081-46d0-baa0-87d6248c987b", 5*time.Second, "echo", []string{"test"})
	a2 := actions.NewProcessAction("84140ab2-612d-4d93-9360-162a4bd5de14", 5*time.Second, "echo", []string{"test2"})

	err := cr.StartAction(a1)
	require.NoError(t, err)

	err = cr.StartAction(a2)
	require.NoError(t, err)

	expected := []*agentv1.ActionResultRequest{
		{ActionId: "6a479303-5081-46d0-baa0-87d6248c987b", Output: []byte("test\n"), Done: true},
		{ActionId: "84140ab2-612d-4d93-9360-162a4bd5de14", Output: []byte("test2\n"), Done: true},
	}
	assertActionResults(t, cr, expected...)
	cr.wg.Wait()
	assert.Empty(t, cr.cancels)
}

func TestCapacityLimit(t *testing.T) {
	t.Parallel()

	cr := New(2, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)

	j1 := testJob{id: "test-1", timeout: time.Second}
	j2 := testJob{id: "test-2", timeout: time.Second}
	j3 := testJob{id: "test-3", timeout: 2 * time.Second}
	j4 := testJob{id: "test-4", timeout: 2 * time.Second}

	require.NoError(t, cr.StartJob(j1))
	require.NoError(t, cr.StartJob(j2))

	// Let first and second jobs start
	time.Sleep(200 * time.Millisecond)

	require.NoError(t, cr.StartJob(j3))
	require.NoError(t, cr.StartJob(j4))

	// Let third and forth jobs to reach semaphores
	time.Sleep(300 * time.Millisecond)

	// First two jobs are started
	assert.True(t, cr.IsRunning(j1.ID()))
	assert.True(t, cr.IsRunning(j2.ID()))
	assert.False(t, cr.IsRunning(j3.ID()))
	assert.False(t, cr.IsRunning(j4.ID()))

	time.Sleep(time.Second)

	// After one more second job terminated and third started
	assert.False(t, cr.IsRunning(j1.ID()))
	assert.False(t, cr.IsRunning(j2.ID()))
	assert.True(t, cr.IsRunning(j3.ID()))
	assert.True(t, cr.IsRunning(j4.ID()))

	time.Sleep(2 * time.Second)

	// After two seconds all jobs are terminated
	assert.False(t, cr.IsRunning(j1.ID()))
	assert.False(t, cr.IsRunning(j2.ID()))
	assert.False(t, cr.IsRunning(j3.ID()))
	assert.False(t, cr.IsRunning(j4.ID()))
}

func TestDefaultCapacityLimit(t *testing.T) {
	t.Parallel()

	// Use default capacity
	cr := New(0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)

	totalJobs := 2 * defaultTotalCapacity
	for i := 0; i < totalJobs; i++ {
		require.NoError(t, cr.StartJob(testJob{id: fmt.Sprintf("test-%d", i), timeout: time.Second}))
	}

	// Let jobs to start
	time.Sleep(500 * time.Millisecond)

	var running int
	for i := 0; i < totalJobs; i++ {
		// Check that running jobs amount is not exceeded default capacity.
		if cr.IsRunning(fmt.Sprintf("test-%d", i)) {
			running++
		}
	}

	assert.Equal(t, defaultTotalCapacity, running)
}

func TestPerDBInstanceLimit(t *testing.T) {
	t.Parallel()

	cr := New(10, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)

	db1j1 := testJob{id: "test-1", timeout: time.Second, dsn: "postgresql://db1"}
	db1j2 := testJob{id: "test-2", timeout: time.Second, dsn: "postgresql://db1"}
	db1j3 := testJob{id: "test-3", timeout: time.Second, dsn: "postgresql://db1"}
	db2j1 := testJob{id: "test-4", timeout: time.Second, dsn: "postgresql://db2"}
	db2j2 := testJob{id: "test-5", timeout: time.Second, dsn: "postgresql://db2"}
	db2j3 := testJob{id: "test-6", timeout: time.Second, dsn: "postgresql://db2"}

	require.NoError(t, cr.StartJob(db1j1), "start job db1j1 failed")
	require.NoError(t, cr.StartJob(db2j1), "start job db2j1 failed")

	// Let jobs to start
	time.Sleep(200 * time.Millisecond)

	require.NoError(t, cr.StartJob(db1j2), "start job db1j2 failed")
	require.NoError(t, cr.StartJob(db2j2), "start job db2j2 failed")
	require.NoError(t, cr.StartJob(db1j3), "start job db1j3 failed")
	require.NoError(t, cr.StartJob(db2j3), "start job db2j3 failed")

	// Let rest jobs to reach semaphores
	time.Sleep(300 * time.Millisecond)

	assert.True(t, cr.IsRunning(db1j1.ID()), "db1j1 is not running")
	assert.True(t, cr.IsRunning(db2j1.ID()), "db2j1 is not running")
	assert.False(t, cr.IsRunning(db1j2.ID()), "db1j2 is running")
	assert.False(t, cr.IsRunning(db2j2.ID()), "db2j2 is running")
	assert.False(t, cr.IsRunning(db1j3.ID()), "db1j3 is running")
	assert.False(t, cr.IsRunning(db2j3.ID()), "db2j3 is running")

	// Over time all jobs are terminated
	time.Sleep(3 * time.Second)

	assert.False(t, cr.IsRunning(db1j1.ID()), "db1j1 is running")
	assert.False(t, cr.IsRunning(db2j1.ID()), "db2j1 is running")
	assert.False(t, cr.IsRunning(db1j2.ID()), "db1j2 is running")
	assert.False(t, cr.IsRunning(db2j2.ID()), "db2j2 is running")
	assert.False(t, cr.IsRunning(db1j3.ID()), "db1j3 is running")
	assert.False(t, cr.IsRunning(db2j3.ID()), "db2j3 is running")
}

func TestDefaultPerDBInstanceLimit(t *testing.T) {
	t.Parallel()

	cr := New(10, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)

	db1j1 := testJob{id: "test-1", timeout: time.Second, dsn: "postgresql://db1"}
	db1j2 := testJob{id: "test-2", timeout: time.Second, dsn: "postgresql://db1"}
	db1j3 := testJob{id: "test-3", timeout: time.Second, dsn: "postgresql://db1"}
	db2j1 := testJob{id: "test-4", timeout: time.Second, dsn: "postgresql://db2"}
	db2j2 := testJob{id: "test-5", timeout: time.Second, dsn: "postgresql://db2"}
	db2j3 := testJob{id: "test-6", timeout: time.Second, dsn: "postgresql://db2"}

	require.NoError(t, cr.StartJob(db1j1))
	require.NoError(t, cr.StartJob(db2j1))
	require.NoError(t, cr.StartJob(db1j2))
	require.NoError(t, cr.StartJob(db2j2))

	// Let jobs to start
	time.Sleep(200 * time.Millisecond)

	require.NoError(t, cr.StartJob(db1j3))
	require.NoError(t, cr.StartJob(db2j3))

	// Let rest jobs to reach semaphores
	time.Sleep(300 * time.Millisecond)

	assert.True(t, cr.IsRunning(db1j1.ID()))
	assert.True(t, cr.IsRunning(db2j1.ID()))
	assert.True(t, cr.IsRunning(db1j2.ID()))
	assert.True(t, cr.IsRunning(db2j2.ID()))
	assert.False(t, cr.IsRunning(db1j3.ID()))
	assert.False(t, cr.IsRunning(db2j3.ID()))

	// Over time all jobs are terminated
	time.Sleep(2 * time.Second)

	assert.False(t, cr.IsRunning(db1j1.ID()))
	assert.False(t, cr.IsRunning(db2j1.ID()))
	assert.False(t, cr.IsRunning(db1j2.ID()))
	assert.False(t, cr.IsRunning(db2j2.ID()))
	assert.False(t, cr.IsRunning(db1j3.ID()))
	assert.False(t, cr.IsRunning(db2j3.ID()))
}

func TestConcurrentRunnerTimeout(t *testing.T) {
	t.Parallel()
	cr := New(0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)
	a1 := actions.NewProcessAction("6a479303-5081-46d0-baa0-87d6248c987b", time.Second, "sleep", []string{"20"})
	a2 := actions.NewProcessAction("84140ab2-612d-4d93-9360-162a4bd5de14", time.Second, "sleep", []string{"30"})

	err := cr.StartAction(a1)
	require.NoError(t, err)

	err = cr.StartAction(a2)
	require.NoError(t, err)

	// https://github.com/golang/go/issues/21880
	expected := []*agentv1.ActionResultRequest{
		{ActionId: "6a479303-5081-46d0-baa0-87d6248c987b", Output: []byte{}, Error: "signal: killed", Done: true},
		{ActionId: "84140ab2-612d-4d93-9360-162a4bd5de14", Output: []byte{}, Error: "signal: killed", Done: true},
	}
	assertActionResults(t, cr, expected...)
	cr.wg.Wait()
	assert.Empty(t, cr.cancels)
}

func TestConcurrentRunnerStop(t *testing.T) {
	t.Parallel()
	cr := New(0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)
	a1 := actions.NewProcessAction("6a479303-5081-46d0-baa0-87d6248c987b", 5*time.Second, "sleep", []string{"20"})
	a2 := actions.NewProcessAction("84140ab2-612d-4d93-9360-162a4bd5de14", 5*time.Second, "sleep", []string{"30"})

	err := cr.StartAction(a1)
	require.NoError(t, err)

	err = cr.StartAction(a2)
	require.NoError(t, err)

	time.Sleep(time.Second)

	cr.Stop(a1.ID())
	cr.Stop(a2.ID())

	// https://github.com/golang/go/issues/21880
	expected := []*agentv1.ActionResultRequest{
		{ActionId: "6a479303-5081-46d0-baa0-87d6248c987b", Output: []byte{}, Error: "signal: killed", Done: true},
		{ActionId: "84140ab2-612d-4d93-9360-162a4bd5de14", Output: []byte{}, Error: "signal: killed", Done: true},
	}
	assertActionResults(t, cr, expected...)
	cr.wg.Wait()
	assert.Empty(t, cr.cancels)
}

func TestConcurrentRunnerCancel(t *testing.T) {
	t.Parallel()
	cr := New(0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	go cr.Run(ctx)

	a1 := actions.NewProcessAction("6a479303-5081-46d0-baa0-87d6248c987b", 5*time.Second, "sleep", []string{"20"})
	a2 := actions.NewProcessAction("84140ab2-612d-4d93-9360-162a4bd5de14", 5*time.Second, "sleep", []string{"30"})

	err := cr.StartAction(a1)
	require.NoError(t, err)

	err = cr.StartAction(a2)
	require.NoError(t, err)

	time.Sleep(time.Second) // To let actions to actually start
	cancel()

	// Unlike other tests, there we mostly see "context canceled", but "signal: killed" still happens.
	// Check both.
	actual := make([]agentv1.AgentRequestPayload, 2)
	actual[0] = <-cr.ActionsResults()
	actual[1] = <-cr.ActionsResults()
	sort.Slice(actual, func(i, j int) bool {
		return actual[i].(*agentv1.ActionResultRequest).ActionId < actual[j].(*agentv1.ActionResultRequest).ActionId
	})
	assert.Equal(t, "6a479303-5081-46d0-baa0-87d6248c987b", actual[0].(*agentv1.ActionResultRequest).ActionId)
	assert.Contains(t, []string{"signal: killed", context.Canceled.Error()}, actual[0].(*agentv1.ActionResultRequest).Error)
	assert.True(t, actual[0].(*agentv1.ActionResultRequest).Done)
	assert.Equal(t, "84140ab2-612d-4d93-9360-162a4bd5de14", actual[1].(*agentv1.ActionResultRequest).ActionId)
	assert.Contains(t, []string{"signal: killed", context.Canceled.Error()}, actual[1].(*agentv1.ActionResultRequest).Error)
	assert.True(t, actual[1].(*agentv1.ActionResultRequest).Done)
	cr.wg.Wait()
	assert.Empty(t, cr.cancels)
}

func TestSemaphoresReleasing(t *testing.T) {
	t.Parallel()
	cr := New(1, 1)
	err := cr.gSem.Acquire(context.TODO(), 1) // Acquire global semaphore to block all jobs
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cr.Run(ctx)

	j := testJob{id: "test-1", timeout: time.Second, dsn: "test"}

	require.NoError(t, cr.StartJob(j))

	// Let job to start
	time.Sleep(200 * time.Millisecond)

	// Check that job is started and local semaphore was acquired
	assert.Equal(t, 1, cr.lSemsLen())

	// Check that job is not running, because it's waiting for global semaphore to be acquired
	assert.False(t, cr.IsRunning(j.ID()))

	// Cancel context to stop job
	cancel()

	// Let job to start and release resources
	time.Sleep(200 * time.Millisecond)

	// Check that local samaphore was released
	assert.Zero(t, cr.lSemsLen())
}

type testJob struct {
	id      string
	timeout time.Duration
	dsn     string
}

func (t testJob) ID() string {
	return t.id
}

func (t testJob) Type() jobs.JobType {
	return jobs.JobType("test")
}

func (t testJob) Timeout() time.Duration {
	return t.timeout
}

func (t testJob) DSN() string {
	return t.dsn
}

func (t testJob) Run(ctx context.Context, send jobs.Send) error { //nolint:revive
	<-ctx.Done()
	return nil
}
