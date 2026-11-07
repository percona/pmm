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

package mcp

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const vmProxy = "/graph/api/datasources/proxy/uid/metrics-uid/"

var testAuth = callerAuth{authorization: testToken}

func queryOf(t *testing.T, r recordedRequest) url.Values {
	t.Helper()

	v, err := url.ParseQuery(r.query)
	require.NoError(t, err)
	return v
}

func keysOf(v url.Values) []string {
	out := make([]string, 0, len(v))
	for k := range v {
		out = append(out, k)
	}
	return out
}

func TestClientDebugOff(t *testing.T) {
	// SWAGGER_DEBUG and DEBUG are what httptransport.New reads for its debug setting.
	t.Setenv("SWAGGER_DEBUG", "1")
	t.Setenv("DEBUG", "1")

	c, err := newClient("http://127.0.0.1:1/")
	require.NoError(t, err)
	assert.False(t, c.swagger.Debug, "go-openapi's debug dump would write the caller's Authorization and cookies to its logger")
}

func TestPromRequestArguments(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, map[string]fixture{
		"GET " + vmProxy + "api/v1/query":       {body: []byte(`{"status":"success","data":{"result":[]}}`)},
		"GET " + vmProxy + "api/v1/query_range": {body: []byte(`{"status":"success","data":{"result":[]}}`)},
	})
	c, err := newClient(fake.server.URL + "/")
	require.NoError(t, err)

	at := time.Unix(1_800_000_000, 0)

	_, err = c.QueryInstant(t.Context(), testAuth, "metrics-uid", promRequest{Query: "up", Time: at, Timeout: metricsTimeout})
	require.NoError(t, err)
	_, err = c.QueryInstant(t.Context(), testAuth, "metrics-uid", promRequest{Query: "up", Timeout: metricsTimeout})
	require.NoError(t, err)
	_, err = c.QueryRange(t.Context(), testAuth, "metrics-uid", promRequest{
		Query: "rate(x[5m])", Start: at.Add(-time.Hour), End: at, Step: 90*time.Second + 500*time.Millisecond, Timeout: metricsTimeout,
	})
	require.NoError(t, err)

	instant := fake.requestsTo(vmProxy + "api/v1/query")
	require.Len(t, instant, 2)
	withTime := queryOf(t, instant[0])
	assert.ElementsMatch(t, []string{"query", "time", "timeout"}, keysOf(withTime))
	assert.Equal(t, "1800000000", withTime.Get("time"))
	assert.Equal(t, "25s", withTime.Get("timeout"))

	noTime := queryOf(t, instant[1])
	assert.ElementsMatch(t, []string{"query", "timeout"}, keysOf(noTime))
	assert.Equal(t, "up", noTime.Get("query"))

	rng := queryOf(t, fake.requestsTo(vmProxy + "api/v1/query_range")[0])
	assert.ElementsMatch(t, []string{"query", "start", "end", "step", "timeout"}, keysOf(rng))
	assert.Equal(t, "1799996400", rng.Get("start"))
	assert.Equal(t, "1800000000", rng.Get("end"))
	assert.Equal(t, "90", rng.Get("step"), "whole seconds")
	assert.Equal(t, "25s", rng.Get("timeout"))

	assert.Empty(t, promRequest{}.values())
}

func TestQueryRangeDecodes(t *testing.T) {
	t.Parallel()

	body := `{"status":"success","data":{"resultType":"matrix","result":[` +
		`{"metric":{"service_name":"db1","__name__":"m"},"values":[[1800000000,"1.5"],[1800000060.5,"NaN"],[1800000120,"+Inf"]]},` +
		`{"metric":{"service_name":"db2"},"values":[]}]}}`
	fake := newFakePMM(t, map[string]fixture{"GET " + vmProxy + "api/v1/query_range": {body: []byte(body)}})
	c, err := newClient(fake.server.URL + "/")
	require.NoError(t, err)

	got, err := c.QueryRange(t.Context(), testAuth, "metrics-uid", promRequest{Query: "m"})
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, map[string]string{"service_name": "db1", "__name__": "m"}, got[0].Labels)
	require.Len(t, got[0].Points, 3)
	assert.Equal(t, time.Unix(1_800_000_000, 0).UTC(), got[0].Points[0].T)
	assert.InDelta(t, 1.5, got[0].Points[0].V, 0)
	assert.Equal(t, time.Unix(1_800_000_060, 500_000_000).UTC(), got[0].Points[1].T)
	assert.True(t, math.IsNaN(got[0].Points[1].V), "NaN is kept, not dropped or zeroed")
	assert.True(t, math.IsInf(got[0].Points[2].V, 1))
	assert.Empty(t, got[1].Points)
}

func TestListCallsArguments(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, map[string]fixture{
		"GET " + vmProxy + "api/v1/label/__name__/values": {body: []byte(`{"status":"success","data":["a","b"]}`)},
		"GET " + vmProxy + "api/v1/labels":                {body: []byte(`{"status":"success","data":["job","service_name"]}`)},
		"GET " + vmProxy + "api/v1/series":                {body: []byte(`{"status":"success","data":[{"__name__":"a","service_name":"db1"}]}`)},
	})
	c, err := newClient(fake.server.URL + "/")
	require.NoError(t, err)

	start, end := time.Unix(1_800_000_000, 0), time.Unix(1_800_003_600, 0)
	match := []string{`up{job="x"}`, `{__name__=~"mysql_.+"}`}

	values, err := c.LabelValues(t.Context(), testAuth, "metrics-uid", "__name__", match, start, end, 50)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, values)
	labels, err := c.Labels(t.Context(), testAuth, "metrics-uid", match, start, end, 50)
	require.NoError(t, err)
	assert.Equal(t, []string{"job", "service_name"}, labels)
	series, err := c.Series(t.Context(), testAuth, "metrics-uid", match, start, end, 2001)
	require.NoError(t, err)
	assert.Equal(t, []map[string]string{{"__name__": "a", "service_name": "db1"}}, series)

	for path, limit := range map[string]string{
		vmProxy + "api/v1/label/__name__/values": "50",
		vmProxy + "api/v1/labels":                "50",
		vmProxy + "api/v1/series":                "2001",
	} {
		reqs := fake.requestsTo(path)
		require.Len(t, reqs, 1, path)
		q := queryOf(t, reqs[0])
		assert.ElementsMatch(t, []string{"match[]", "start", "end", "limit"}, keysOf(q), path)
		assert.Equal(t, match, q["match[]"], path)
		assert.Equal(t, "1800000000", q.Get("start"), path)
		assert.Equal(t, "1800003600", q.Get("end"), path)
		assert.Equal(t, limit, q.Get("limit"), path)
	}
}

func TestMapError422(t *testing.T) {
	t.Parallel()

	te := mapError(&statusError{status: http.StatusUnprocessableEntity, method: "GET", path: "/x", message: "cannot parse query: unexpected token"})
	assert.Equal(t, codeInvalidInput, te.code)
	assert.Contains(t, te.message, "cannot parse query: unexpected token")

	fake := newFakePMM(t, map[string]fixture{
		"GET " + vmProxy + "api/v1/query": {status: http.StatusUnprocessableEntity, body: []byte("cannot parse query\n")},
	})
	c, err := newClient(fake.server.URL + "/")
	require.NoError(t, err)
	_, err = c.QueryInstant(t.Context(), testAuth, "metrics-uid", promRequest{Query: "up{"})
	require.Error(t, err)
	assert.Equal(t, codeInvalidInput, mapError(err).code)
	assert.Contains(t, mapError(err).message, "cannot parse query")
}

func TestNullable(t *testing.T) {
	t.Parallel()

	assert.Nil(t, nullable(math.NaN()))
	assert.Nil(t, nullable(math.Inf(1)))
	assert.Nil(t, nullable(math.Inf(-1)))
	for _, v := range []float64{0, -1.5, math.MaxFloat64} {
		got := nullable(v)
		require.NotNil(t, got)
		assert.InDelta(t, v, *got, 0)
	}
	assert.InDelta(t, 0, finite(lenientFloat(math.NaN())), 0)
	assert.InDelta(t, 7, finite(7), 0)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type countingBody struct {
	r io.Reader
	n *atomic.Int64
}

func (b countingBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.n.Add(int64(n))
	return n, err
}

func (countingBody) Close() error { return nil }

func TestVMResponseBounded(t *testing.T) {
	t.Parallel()

	c, err := newClient("http://pmm.test/")
	require.NoError(t, err)

	var read atomic.Int64
	c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := io.MultiReader(strings.NewReader(`{"status":"success","data":[`), bytes.NewReader(bytes.Repeat([]byte(" "), 5<<20)))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: countingBody{r: body, n: &read}}, nil
	})}

	_, err = c.Series(t.Context(), testAuth, "metrics-uid", []string{"up"}, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errResponseTooLarge)
	assert.LessOrEqual(t, read.Load(), int64(maxVMResponseBytes+1))
	assert.Equal(t, codePMMUnavailable, mapError(err).code)

	read.Store(0)
	c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := strings.NewReader(`{"status":"success","data":["a"]}`)
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: countingBody{r: body, n: &read}}, nil
	})}
	got, err := c.Labels(t.Context(), testAuth, "metrics-uid", nil, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, got)
}

func TestVMRequestLimit(t *testing.T) {
	// Not parallel: it sets the environment the client reads its limit from.
	t.Setenv("VM_search_maxConcurrentRequests", "4")

	release := make(chan struct{})
	var arrived atomic.Int32
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		arrived.Add(1)
		<-release
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"status":"success","data":{"result":[]}}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { once.Do(func() { close(release) }) })

	c, err := newClient(srv.URL + "/")
	require.NoError(t, err)
	require.Equal(t, 2, cap(c.vmSlots), "half of the configured limit")

	call := func(ctx context.Context) error {
		_, err := c.QueryInstant(ctx, testAuth, "metrics-uid", promRequest{Query: "up"})
		return err
	}

	errs := make(chan error, 3)
	for range 2 {
		go func() { errs <- call(t.Context()) }()
	}
	require.Eventually(t, func() bool { return arrived.Load() == 2 }, 5*time.Second, 5*time.Millisecond)

	go func() { errs <- call(t.Context()) }()
	time.Sleep(200 * time.Millisecond)
	assert.EqualValues(t, 2, arrived.Load(), "the third request waits for a slot")

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	err = call(ctx)
	require.Error(t, err)
	assert.Equal(t, codeTimeout, mapError(err).code)
	assert.EqualValues(t, 2, arrived.Load(), "a request that gave up sent nothing")

	once.Do(func() { close(release) })
	for range 3 {
		require.NoError(t, <-errs)
	}
	assert.EqualValues(t, 3, arrived.Load(), "the waiting request ran once a slot was free")
}

func TestVMRequestLimitDefault(t *testing.T) {
	t.Setenv("VM_search_maxConcurrentRequests", "")
	assert.Equal(t, 4, maxVMRequestsPerReplica())

	t.Setenv("VM_search_maxConcurrentRequests", "1")
	assert.Equal(t, 1, maxVMRequestsPerReplica(), "at least one slot")

	t.Setenv("VM_search_maxConcurrentRequests", "16")
	assert.Equal(t, 8, maxVMRequestsPerReplica())
}

func TestLabelValuesRefusesAPathLabel(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, map[string]fixture{
		"GET " + vmProxy + "api/v1/label/mysql_up/values": {body: []byte(`{"status":"success","data":["a"]}`)},
	})
	c, err := newClient(fake.server.URL + "/")
	require.NoError(t, err)

	for _, label := range []string{"..", "a/b", "../query", "", "1a", "a b", "a%2Fb"} {
		_, err = c.LabelValues(t.Context(), testAuth, "metrics-uid", label, nil, time.Time{}, time.Time{}, 0)
		require.Error(t, err, label)
		assert.Equal(t, codeInvalidInput, mapError(err).code, label)
	}
	assert.Zero(t, fake.requestCount(), "a refused label sends no request")

	got, err := c.LabelValues(t.Context(), testAuth, "metrics-uid", "mysql_up", nil, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, got)
	assert.Equal(t, 1, fake.requestCount())
}
