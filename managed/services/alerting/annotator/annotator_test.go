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

package annotator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type createCall struct {
	tags  []string
	start time.Time
	text  string
}

type endCall struct {
	id  int
	end time.Time
}

type fakeClient struct {
	findID  int
	created []createCall
	ends    []endCall
}

func (f *fakeClient) CreateAlertAnnotation(_ context.Context, tags []string, start time.Time, text string) (int, error) {
	f.created = append(f.created, createCall{tags: tags, start: start, text: text})
	return 99, nil
}

func (f *fakeClient) SetAlertAnnotationEnd(_ context.Context, id int, end time.Time) error {
	f.ends = append(f.ends, endCall{id: id, end: end})
	return nil
}

func (f *fakeClient) FindAlertAnnotationID(_ context.Context, _ []string, _, _ time.Time) (int, error) {
	return f.findID, nil
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return ts
}

func TestProcessAlert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	start := mustTime(t, "2026-06-13T10:00:00Z")
	end := mustTime(t, "2026-06-13T10:05:00Z")

	mysqlDown := webhookAlert{
		Status:      "firing",
		Fingerprint: "abc",
		StartsAt:    start,
		Labels:      map[string]string{"service_name": "mysql-1", "node_name": "node-1", "alertname": "MySQL down", "severity": "critical"},
		Annotations: map[string]string{"summary": "MySQL down (mysql-1)"},
	}

	t.Run("firing creates a scoped annotation", func(t *testing.T) {
		t.Parallel()
		f := &fakeClient{findID: 0}
		s := New(f)

		require.NoError(t, s.processAlert(ctx, mysqlDown))

		require.Len(t, f.created, 1)
		assert.Empty(t, f.ends)
		c := f.created[0]
		assert.Equal(t, start, c.start)
		assert.Equal(t, "MySQL down (mysql-1)", c.text)
		assert.Contains(t, c.tags, "mysql-1")
		assert.Contains(t, c.tags, "node-1")
		assert.Contains(t, c.tags, "pmm_alert_fingerprint:abc")
		assert.Contains(t, c.tags, "alertname:MySQL down")
		assert.Contains(t, c.tags, "severity:critical")
		assert.NotContains(t, c.tags, "pmm_annotation", "service-scoped alert must not get the global tag")
	})

	t.Run("firing is deduplicated when already annotated", func(t *testing.T) {
		t.Parallel()
		f := &fakeClient{findID: 42}
		s := New(f)

		require.NoError(t, s.processAlert(ctx, mysqlDown))
		assert.Empty(t, f.created)
		assert.Empty(t, f.ends)
	})

	t.Run("resolved closes the existing annotation into a region", func(t *testing.T) {
		t.Parallel()
		f := &fakeClient{findID: 42}
		s := New(f)

		resolved := mysqlDown
		resolved.Status = "resolved"
		resolved.EndsAt = end

		require.NoError(t, s.processAlert(ctx, resolved))
		assert.Empty(t, f.created)
		require.Len(t, f.ends, 1)
		assert.Equal(t, 42, f.ends[0].id)
		assert.Equal(t, end, f.ends[0].end)
	})

	t.Run("resolved with missing start creates then closes", func(t *testing.T) {
		t.Parallel()
		f := &fakeClient{findID: 0}
		s := New(f)

		resolved := mysqlDown
		resolved.Status = "resolved"
		resolved.EndsAt = end

		require.NoError(t, s.processAlert(ctx, resolved))
		require.Len(t, f.created, 1)
		require.Len(t, f.ends, 1)
		assert.Equal(t, 99, f.ends[0].id) // id returned by CreateAlertAnnotation
		assert.Equal(t, end, f.ends[0].end)
	})

	t.Run("alert without service/node falls back to global tag", func(t *testing.T) {
		t.Parallel()
		f := &fakeClient{findID: 0}
		s := New(f)

		generic := webhookAlert{Status: "firing", Fingerprint: "xyz", StartsAt: start, Labels: map[string]string{"alertname": "Custom"}}
		require.NoError(t, s.processAlert(ctx, generic))
		require.Len(t, f.created, 1)
		assert.Contains(t, f.created[0].tags, "pmm_annotation")
	})
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("rejects non-POST", func(t *testing.T) {
		t.Parallel()
		s := New(&fakeClient{})
		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/internal/webhook", nil))
		assert.Equal(t, http.StatusMethodNotAllowed, rw.Code)
	})

	t.Run("rejects malformed body", func(t *testing.T) {
		t.Parallel()
		s := New(&fakeClient{})
		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, httptest.NewRequest(http.MethodPost, "/internal/webhook", strings.NewReader("not json")))
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("processes a firing payload", func(t *testing.T) {
		t.Parallel()
		f := &fakeClient{findID: 0}
		s := New(f)

		body := `{"alerts":[{"status":"firing","fingerprint":"abc","startsAt":"2026-06-13T10:00:00Z",` +
			`"labels":{"service_name":"mysql-1","node_name":"node-1","alertname":"MySQL down"},` +
			`"annotations":{"summary":"MySQL down (mysql-1)"}}]}`
		rw := httptest.NewRecorder()
		s.ServeHTTP(rw, httptest.NewRequest(http.MethodPost, "/internal/webhook", strings.NewReader(body)))

		assert.Equal(t, http.StatusOK, rw.Code)
		require.Len(t, f.created, 1)
		assert.Equal(t, "MySQL down (mysql-1)", f.created[0].text)
		assert.Contains(t, f.created[0].tags, "mysql-1")
	})
}
