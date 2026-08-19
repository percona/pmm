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

package vmretention

import (
	"context"
	"errors"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setup(t *testing.T, dataRetention time.Duration) *reform.DB {
	t.Helper()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{DataRetention: dataRetention})
	require.NoError(t, err)

	return db
}

func conflictErr() error {
	return apierrors.NewConflict(
		schema.GroupResource{Group: "operator.victoriametrics.com", Resource: "vmclusters"},
		"pmm-vmcluster",
		errors.New("the object has been modified"),
	)
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("PatchesWhenDifferent", func(t *testing.T) {
		db := setup(t, 7*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return(Retention{Period: "90d", resourceVersion: "42"}, nil)
		client.On("Set", ctx, Retention{Period: "7d", resourceVersion: "42"}).Return(nil)

		require.NoError(t, New(db, client).reconcile(ctx))
	})

	t.Run("SkipsWhenEqual", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return(Retention{Period: "30d", resourceVersion: "42"}, nil)
		// Set is deliberately not expected: mockery fails the test if it is called.

		require.NoError(t, New(db, client).reconcile(ctx))
	})

	t.Run("PatchesWhenUnset", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return(Retention{resourceVersion: "7"}, nil)
		client.On("Set", ctx, Retention{Period: "30d", resourceVersion: "7"}).Return(nil)

		require.NoError(t, New(db, client).reconcile(ctx))
	})
}

// TestReconcileConflict covers the optimistic-concurrency path. The VictoriaMetrics operator
// writes status while it rolls vmstorage, so a conflict is most likely right after our own
// write, which makes this the path that matters in practice rather than an edge case.
func TestReconcileConflict(t *testing.T) {
	ctx := context.Background()

	t.Run("RetriesAndSucceeds", func(t *testing.T) {
		db := setup(t, 5*24*time.Hour)
		client := NewMockClient(t)

		// First read is stale and the write conflicts; the retry re-reads and succeeds.
		client.On("Get", ctx).Return(Retention{Period: "90d", resourceVersion: "1"}, nil).Once()
		client.On("Set", ctx, Retention{Period: "5d", resourceVersion: "1"}).Return(conflictErr()).Once()
		client.On("Get", ctx).Return(Retention{Period: "90d", resourceVersion: "2"}, nil).Once()
		client.On("Set", ctx, Retention{Period: "5d", resourceVersion: "2"}).Return(nil).Once()

		require.NoError(t, New(db, client).reconcile(ctx))
	})

	t.Run("ExhaustedReturnsConflict", func(t *testing.T) {
		db := setup(t, 5*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return(Retention{Period: "90d", resourceVersion: "1"}, nil)
		client.On("Set", ctx, mock.Anything).Return(conflictErr())

		err := New(db, client).reconcile(ctx)
		require.Error(t, err)
		// Still recognizable as a conflict, which is what keeps it out of the log throttle.
		assert.True(t, apierrors.IsConflict(err), "expected a conflict, got %v", err)
	})
}

func TestReconcileErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("ReadError", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return(Retention{}, errors.New("forbidden"))

		err := New(db, client).reconcile(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read the current retention period")
	})

	t.Run("WriteError", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return(Retention{Period: "90d", resourceVersion: "1"}, nil)
		client.On("Set", ctx, mock.Anything).Return(errors.New("forbidden"))

		err := New(db, client).reconcile(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to set retention period to "30d"`)
	})
}

// TestMetrics checks that the metrics are absent when VictoriaMetrics is not managed by an
// operator. A permanently zero success gauge on every non-Kubernetes deployment would read
// as a failing reconcile.
func TestMetrics(t *testing.T) {
	collect := func(svc *Service) int {
		ch := make(chan prom.Metric, 10)
		svc.Collect(ch)
		close(ch)

		var n int
		for range ch {
			n++
		}
		return n
	}

	t.Run("SilentWhenDisabled", func(t *testing.T) {
		assert.Equal(t, 0, collect(New(nil, nil)))
	})

	// An HA node that is not the leader never reconciles, so it must not report a standing
	// zero either: with three nodes that would leave two permanently looking like failures.
	t.Run("SilentBeforeFirstReconcile", func(t *testing.T) {
		assert.Equal(t, 0, collect(New(nil, NewMockClient(t))))
	})

	t.Run("ReportedAfterReconcile", func(t *testing.T) {
		db := setup(t, 14*24*time.Hour)
		client := NewMockClient(t)
		// reconcileWithTimeout wraps the context, so it cannot be matched exactly here.
		client.On("Get", mock.Anything).Return(Retention{Period: "90d", resourceVersion: "1"}, nil)
		client.On("Set", mock.Anything, Retention{Period: "14d", resourceVersion: "1"}).Return(nil)

		svc := New(db, client)
		svc.reconcileWithTimeout(context.Background())

		assert.Equal(t, 3, collect(svc))
		assert.InDelta(t, 1, testutilGauge(t, svc.mSuccess), 0)
		assert.InDelta(t, (14 * 24 * time.Hour).Seconds(), testutilGauge(t, svc.mRetention), 0)
	})

	t.Run("FailureIsReported", func(t *testing.T) {
		db := setup(t, 14*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", mock.Anything).Return(Retention{}, errors.New("forbidden"))

		svc := New(db, client)
		svc.reconcileWithTimeout(context.Background())

		assert.InDelta(t, 0, testutilGauge(t, svc.mSuccess), 0)
		// The first failure is recorded so that a repeat can be demoted to debug.
		assert.NotEmpty(t, svc.lastError)
	})
}

func testutilGauge(t *testing.T, g prom.Gauge) float64 {
	t.Helper()

	ch := make(chan prom.Metric, 1)
	g.Collect(ch)
	close(ch)

	m := <-ch
	var pb dto.Metric
	require.NoError(t, m.Write(&pb))
	return pb.GetGauge().GetValue()
}

func TestRunWithoutClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	svc := New(nil, nil)
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	svc.RequestRetentionUpdate() // must not panic on a nil db
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the context was canceled")
	}
}

func TestRetentionPeriod(t *testing.T) {
	for _, tc := range []struct {
		retention time.Duration
		expected  string
	}{
		{24 * time.Hour, "1d"},
		{30 * 24 * time.Hour, "30d"},
		{90 * 24 * time.Hour, "90d"},
		{3650 * 24 * time.Hour, "3650d"},
		// Settings validation rejects sub-day values, so this only documents the fallback.
		{36 * time.Hour, "1d"},
	} {
		assert.Equal(t, tc.expected, retentionPeriod(tc.retention))
	}
}

func TestResourceFor(t *testing.T) {
	assert.Equal(t, "vmclusters", resourceFor(KubeParams{Kind: "VMCluster"}))
	assert.Equal(t, "vmsingles", resourceFor(KubeParams{Kind: "VMSingle"}))
	// The override wins for any kind the derivation does not pluralise correctly.
	assert.Equal(t, "vmauths", resourceFor(KubeParams{Kind: "Anything", Resource: "vmauths"}))
}
