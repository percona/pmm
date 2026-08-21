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
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

// setup returns a DB whose settings row reports dataRetention.
//
// Each sqlmock expectation is fulfilled once, so this answers exactly one settings read: a test
// that reconciles twice needs a DB per reconcile.
func setup(t *testing.T, dataRetention time.Duration) *reform.DB {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mock.ExpectClose()
		assert.NoError(t, sqlDB.Close())
	})

	// Settings are stored as a single JSON document, and a duration marshals as nanoseconds.
	// GetSettings fills the rest of the fields with their defaults.
	row := fmt.Sprintf(`{"data_retention":%d}`, dataRetention)
	mock.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow([]byte(row)))

	return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("PatchesWhenDifferent", func(t *testing.T) {
		db := setup(t, 7*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return("90d", nil)
		client.On("Set", ctx, "7d").Return(nil)

		require.NoError(t, New(db, client).reconcile(ctx))
	})

	t.Run("SkipsWhenEqual", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return("30d", nil)
		// Set is deliberately not expected: mockery fails the test if it is called, which is
		// what keeps an unchanged setting from making the operator roll vmstorage.

		require.NoError(t, New(db, client).reconcile(ctx))
	})

	t.Run("PatchesWhenUnset", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return("", nil)
		client.On("Set", ctx, "30d").Return(nil)

		require.NoError(t, New(db, client).reconcile(ctx))
	})
}

func TestReconcileErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("ReadError", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return("", errors.New("forbidden"))

		err := New(db, client).reconcile(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read the current retention period")
	})

	t.Run("WriteError", func(t *testing.T) {
		db := setup(t, 30*24*time.Hour)
		client := NewMockClient(t)
		client.On("Get", ctx).Return("90d", nil)
		client.On("Set", ctx, mock.Anything).Return(errors.New("forbidden"))

		err := New(db, client).reconcile(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to set retention period to "30d"`)
	})
}

// A failed reconcile records the error, which is what the log throttle in
// reconcileWithTimeout keys on to demote an identical repeat to debug.
func TestReconcileRecordsFailure(t *testing.T) {
	client := NewMockClient(t)
	client.On("Get", mock.Anything).Return("", errors.New("forbidden"))

	svc := New(setup(t, 14*24*time.Hour), client)
	svc.reconcileWithTimeout(context.Background())
	assert.NotEmpty(t, svc.lastError)

	// A success clears it again, so the next failure is announced rather than demoted.
	client2 := NewMockClient(t)
	client2.On("Get", mock.Anything).Return("14d", nil)
	svc2 := New(setup(t, 14*24*time.Hour), client2)
	svc2.lastError = "stale"
	svc2.reconcileWithTimeout(context.Background())
	assert.Empty(t, svc2.lastError)
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

func TestNewKubeClient(t *testing.T) {
	const (
		name       = "pmm-vmcluster"
		apiVersion = "operator.victoriametrics.com/v1beta1"
	)

	t.Run("NotNamedOptsOut", func(t *testing.T) {
		client, err := NewKubeClient(KubeParams{APIVersion: apiVersion, Kind: "VMCluster"})
		require.NoError(t, err)
		assert.Nil(t, client, "an unnamed resource must disable reconciliation, not fail")
	})

	t.Run("NoKindOrResource", func(t *testing.T) {
		_, err := NewKubeClient(KubeParams{Name: name, Namespace: "pmm", APIVersion: apiVersion})
		require.Error(t, err)
		// Without this the plural would be derived as "s".
		assert.Contains(t, err.Error(), "PMM_VM_CLUSTER_KIND")
	})

	// ParseGroupVersion accepts both of these and resolves them to the core API group, where a
	// custom resource can never live, so they have to be rejected here rather than a minute later.
	t.Run("UngroupedAPIVersion", func(t *testing.T) {
		for _, ungrouped := range []string{"", "v1beta1"} {
			_, err := NewKubeClient(KubeParams{Name: name, Namespace: "pmm", APIVersion: ungrouped, Kind: "VMCluster"})
			require.Error(t, err, "API version %q", ungrouped)
			assert.Contains(t, err.Error(), "PMM_VM_CLUSTER_API_VERSION", "API version %q", ungrouped)
		}
	})

	// Naming a resource and then not being in a cluster is a misconfiguration, not an opt-out, so
	// it has to fail at startup instead of leaving retention silently unapplied.
	t.Run("NamedButNotInCluster", func(t *testing.T) {
		t.Setenv("KUBERNETES_SERVICE_HOST", "")

		_, err := NewKubeClient(KubeParams{Name: name, Namespace: "pmm", APIVersion: apiVersion, Kind: "VMCluster"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PMM_VM_CLUSTER_NAME")
	})
}

func TestResourceFor(t *testing.T) {
	assert.Equal(t, "vmclusters", resourceFor(KubeParams{Kind: "VMCluster"}))
	assert.Equal(t, "vmsingles", resourceFor(KubeParams{Kind: "VMSingle"}))
	assert.Equal(t, "vmauths", resourceFor(KubeParams{Kind: "VMAuth"}))
}
