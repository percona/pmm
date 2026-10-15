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

package annotations

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStore returns a Store for a test server, reached with the base path and credentials of pmm-managed's VictoriaMetrics URL.
func newTestStore(t *testing.T, handler http.HandlerFunc) *Store {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	base.Path = "/prometheus/"
	base.User = url.UserPassword("u", "p")

	return NewStore(base)
}

func TestStoreWrite(t *testing.T) {
	t.Parallel()

	events := []Event{
		{
			AnnotationID: 7,
			Time:         time.UnixMilli(1700000000123),
			Text:         "deploy",
			Tags:         []string{"a", "b"},
			Labels:       map[string]string{"node_name": "n1", "service_name": "svc", "empty": ""},
		},
		{
			AnnotationID: 9,
			Time:         time.UnixMilli(1700000100456),
			Text:         "reboot",
			Labels:       map[string]string{"node_name": "n2"},
		},
	}

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		var (
			r    *http.Request
			body []byte
		)
		s := newTestStore(t, func(w http.ResponseWriter, req *http.Request) {
			r = req
			body, _ = io.ReadAll(req.Body)
			w.WriteHeader(http.StatusNoContent)
		})

		require.NoError(t, s.Write(t.Context(), events))

		require.NotNil(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/prometheus/api/v1/write", r.URL.Path)
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		user, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "u", user)
		assert.Equal(t, "p", password)

		decoded, err := snappy.Decode(nil, body)
		require.NoError(t, err)
		var req prompb.WriteRequest
		require.NoError(t, req.Unmarshal(decoded))

		expected := []prompb.TimeSeries{
			{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "pmm_annotation_event"},
					{Name: "annotation_id", Value: "7"},
					{Name: "node_name", Value: "n1"},
					{Name: "service_name", Value: "svc"},
					{Name: "tags", Value: "a,b"},
					{Name: "text", Value: "deploy"},
				},
				Samples: []prompb.Sample{{Value: 1700000000123, Timestamp: 1700000000123}},
			},
			{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "pmm_annotation_event"},
					{Name: "annotation_id", Value: "9"},
					{Name: "node_name", Value: "n2"},
					{Name: "text", Value: "reboot"},
				},
				Samples: []prompb.Sample{{Value: 1700000100456, Timestamp: 1700000100456}},
			},
		}
		assert.Equal(t, expected, req.Timeseries)
	})

	t.Run("ServerError", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "storage is down", http.StatusInternalServerError)
		})

		err := s.Write(t.Context(), events)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "storage is down")
	})

	t.Run("NoEvents", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(_ http.ResponseWriter, _ *http.Request) {
			t.Error("unexpected request")
		})

		require.NoError(t, s.Write(t.Context(), nil))
	})
}

func TestStoreDelete(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		var (
			r    *http.Request
			form url.Values
		)
		s := newTestStore(t, func(w http.ResponseWriter, req *http.Request) {
			r = req
			_ = req.ParseForm()
			form = req.PostForm
			w.WriteHeader(http.StatusNoContent)
		})

		require.NoError(t, s.Delete(t.Context(), []int64{7, 9}))

		require.NotNil(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/prometheus/api/v1/admin/tsdb/delete_series", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		user, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "u", user)
		assert.Equal(t, "p", password)
		assert.Equal(t, []string{`pmm_annotation_event{annotation_id=~"7|9"}`}, form["match[]"])
	})

	t.Run("NoIDs", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(_ http.ResponseWriter, _ *http.Request) {
			t.Error("unexpected request")
		})

		require.NoError(t, s.Delete(t.Context(), nil))
		require.NoError(t, s.Delete(t.Context(), []int64{}))
	})

	t.Run("ServerError", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "forbidden here", http.StatusForbidden)
		})

		err := s.Delete(t.Context(), []int64{7})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
		assert.Contains(t, err.Error(), "forbidden here")
	})
}

func TestStoreKeys(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		var r *http.Request
		s := newTestStore(t, func(w http.ResponseWriter, req *http.Request) {
			r = req
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":[`+
				`{"metric":{"annotation_id":"7","node_name":"n1"},"value":[1700000000,"1"]},`+
				`{"metric":{"annotation_id":"9","node_name":"n2","service_name":"svc"},"value":[1700000000,"1"]}]}}`)
		})

		keys, err := s.Keys(t.Context(), 30*24*time.Hour)
		require.NoError(t, err)

		require.NotNil(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/prometheus/api/v1/query", r.URL.Path)
		assert.Equal(t,
			"count by (annotation_id, node_name, service_name) (last_over_time(pmm_annotation_event[2592000s]))",
			r.URL.Query().Get("query"))
		user, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "u", user)
		assert.Equal(t, "p", password)

		assert.Equal(t, []Key{
			{AnnotationID: 7, NodeName: "n1"},
			{AnnotationID: 9, NodeName: "n2", ServiceName: "svc"},
		}, keys)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":[]}}`)
		})

		keys, err := s.Keys(t.Context(), time.Hour)
		require.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("ServerError", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "query timeout", http.StatusServiceUnavailable)
		})

		keys, err := s.Keys(t.Context(), time.Hour)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "503")
		assert.Contains(t, err.Error(), "query timeout")
		assert.Nil(t, keys)
	})

	t.Run("ErrorStatus", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"status":"error","errorType":"bad_data","error":"cannot parse"}`)
		})

		keys, err := s.Keys(t.Context(), time.Hour)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse")
		assert.Nil(t, keys)
	})

	t.Run("InvalidAnnotationID", func(t *testing.T) {
		t.Parallel()

		s := newTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":[`+
				`{"metric":{"annotation_id":"abc","node_name":"n1"},"value":[1700000000,"1"]}]}}`)
		})

		keys, err := s.Keys(t.Context(), time.Hour)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "abc")
		assert.Nil(t, keys)
	})
}

func TestStoreBaseURL(t *testing.T) {
	t.Parallel()

	var r *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r = req
		_, _ = io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":[]}}`)
	}))
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)

	_, err = NewStore(base).Keys(t.Context(), time.Hour)
	require.NoError(t, err)

	require.NotNil(t, r)
	assert.Equal(t, "/api/v1/query", r.URL.Path)
	assert.Empty(t, r.Header.Get("Authorization"))
}
