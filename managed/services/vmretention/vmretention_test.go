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
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
		// Asserted before ExpectClose is queued, or that expectation is itself pending here.
		// Without this the settings read below is optional, and a reconcile that stopped
		// reading settings altogether would still pass.
		assert.NoError(t, mock.ExpectationsWereMet())
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
	ctx := t.Context()

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
	ctx := t.Context()

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
	svc.reconcileWithTimeout(t.Context())
	assert.NotEmpty(t, svc.lastError)

	// A success clears it again, so the next failure is announced rather than demoted.
	client2 := NewMockClient(t)
	client2.On("Get", mock.Anything).Return("14d", nil)
	svc2 := New(setup(t, 14*24*time.Hour), client2)
	svc2.lastError = "stale"
	svc2.reconcileWithTimeout(t.Context())
	assert.Empty(t, svc2.lastError)
}

func TestRunWithoutClient(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	svc := New(nil, nil)
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	// The send is non-blocking, so it must not wedge with nothing draining reloadCh.
	svc.RequestRetentionUpdate()
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the context was canceled")
	}
}

// captureLogs collects what the package logs through the standard logger. The hook is global,
// so callers must not run in parallel.
func captureLogs(t *testing.T) *logrustest.Hook {
	t.Helper()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(logrus.LevelHooks{}) })

	return hook
}

// Only a named resource that could not be reached is a misconfiguration. Naming nothing is how
// most deployments opt out, and saying so at error level would cry wolf on every one of them.
func TestDisabledReasonIsAnnounced(t *testing.T) {
	t.Run("NotNamedIsNotAnError", func(t *testing.T) {
		hook := captureLogs(t)

		New(nil, nil)

		entries := hook.AllEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, logrus.InfoLevel, entries[0].Level)
		// The supervisord program is deleted when VictoriaMetrics is external, so claiming it
		// applies retention would send an operator to a file that is not there.
		assert.NotContains(t, entries[0].Message, "supervisord")
	})

	t.Run("NamedButUnusableIsAnError", func(t *testing.T) {
		hook := captureLogs(t)

		NewDisabled(errors.New("not a group-qualified API version"))

		entries := hook.AllEntries()
		require.Len(t, entries, 1)
		assert.Equal(t, logrus.ErrorLevel, entries[0].Level)
		assert.Contains(t, entries[0].Message, "not a group-qualified API version")
	})
}

// A node promoted later has its own log to explain why retention is not moving, so the reason
// is repeated per leadership term rather than only at start-up.
func TestDisabledServiceReAnnouncesOnPromotion(t *testing.T) {
	svc := NewDisabled(errors.New("not a group-qualified API version"))

	hook := captureLogs(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the context was canceled")
	}

	entries := hook.AllEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, logrus.ErrorLevel, entries[0].Level)
	assert.Contains(t, entries[0].Message, "not a group-qualified API version")
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

	// An empty namespace is accepted by client-go and silently builds a cluster-scoped request
	// path, where a namespaced resource can never be found, so it has to be rejected here.
	t.Run("EmptyNamespaceIsRejected", func(t *testing.T) {
		t.Run("FromTheServiceAccountFile", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "namespace")
			require.NoError(t, os.WriteFile(path, []byte("  \n"), 0o600))

			original := namespaceFile
			t.Cleanup(func() { namespaceFile = original })
			namespaceFile = path

			_, err := NewKubeClient(KubeParams{Name: name, APIVersion: apiVersion, Kind: "VMCluster"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "set PMM_VM_CLUSTER_NAMESPACE")
		})

		t.Run("FromTheEnvironment", func(t *testing.T) {
			_, err := NewKubeClient(KubeParams{Name: name, Namespace: "   ", APIVersion: apiVersion, Kind: "VMCluster"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "set PMM_VM_CLUSTER_NAMESPACE")
		})
	})

	// The projected file is the production path, since the chart only sets
	// PMM_VM_CLUSTER_NAMESPACE when it deploys a VMCluster itself. A real pod carries the bare
	// name with no trailing newline, but TrimSpace is what makes either shape work.
	t.Run("NamespaceFromTheServiceAccountFile", func(t *testing.T) {
		for _, content := range []string{"pmm", "pmm\n"} {
			path := filepath.Join(t.TempDir(), "namespace")
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

			original := namespaceFile
			namespaceFile = path

			_, err := NewKubeClient(KubeParams{Name: name, APIVersion: apiVersion, Kind: "VMCluster"})
			namespaceFile = original

			// Asserting which error arrives is the point: NewKubeClient fails either way outside
			// a cluster, so requiring an error alone would pass even if the guard had wrongly
			// rejected a perfectly good namespace.
			require.Error(t, err, "namespace file %q", content)
			assert.Contains(t, err.Error(), "in-cluster configuration", "namespace file %q", content)
		}
	})

	// Naming a resource and then not being in a cluster is a misconfiguration, not an opt-out, so
	// it has to be reported as an error rather than the nil client that means "not wanted here".
	t.Run("NamedButNotInCluster", func(t *testing.T) {
		t.Setenv("KUBERNETES_SERVICE_HOST", "")

		_, err := NewKubeClient(KubeParams{Name: name, Namespace: "pmm", APIVersion: apiVersion, Kind: "VMCluster"})
		require.Error(t, err)
		// Not on PMM_VM_CLUSTER_NAME: that is also a substring of PMM_VM_CLUSTER_NAMESPACE,
		// so it would pass on the namespace branch this subtest does not exercise.
		assert.Contains(t, err.Error(), "in-cluster configuration")
	})
}

func TestResourceFor(t *testing.T) {
	assert.Equal(t, "vmclusters", resourceFor(KubeParams{Kind: "VMCluster"}))
	assert.Equal(t, "vmsingles", resourceFor(KubeParams{Kind: "VMSingle"}))
	assert.Equal(t, "vmauths", resourceFor(KubeParams{Kind: "VMAuth"}))
}
