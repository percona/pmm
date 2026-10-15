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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

const httpTimeout = 30 * time.Second

// Key identifies an annotation series by annotation and by the node or service it is scoped to.
type Key struct {
	AnnotationID int64
	NodeName     string
	ServiceName  string
}

// Store writes, deletes and lists annotation series in VictoriaMetrics.
type Store struct {
	base   *url.URL
	user   *url.Userinfo
	client *http.Client
}

// NewStore returns a Store for the VictoriaMetrics URL that pmm-managed uses; its userinfo, if any, is sent as basic auth.
func NewStore(base *url.URL) *Store {
	u := *base
	u.User = nil

	return &Store{
		base:   &u,
		user:   base.User,
		client: &http.Client{Timeout: httpTimeout},
	}
}

// Write stores one series per event, with the event time as both the sample timestamp and value.
func (s *Store) Write(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	wr := prompb.WriteRequest{Timeseries: make([]prompb.TimeSeries, 0, len(events))}
	for _, e := range events {
		wr.Timeseries = append(wr.Timeseries, e.timeSeries())
	}

	data, err := wr.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal remote write request: %w", err)
	}

	req, err := s.newRequest(ctx, http.MethodPost, "api/v1/write", bytes.NewReader(snappy.Encode(nil, data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	_, err = s.do(req)

	return err
}

// Delete removes the series of the annotations with the given IDs.
func (s *Store) Delete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = strconv.FormatInt(id, 10)
	}
	form := url.Values{"match[]": {fmt.Sprintf(`%s{annotation_id=~"%s"}`, MetricName, strings.Join(values, "|"))}}

	req, err := s.newRequest(ctx, http.MethodPost, "api/v1/admin/tsdb/delete_series", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err = s.do(req)

	return err
}

// Keys lists the annotation series that have a sample within the window.
func (s *Store) Keys(ctx context.Context, window time.Duration) ([]Key, error) {
	query := fmt.Sprintf("count by (annotation_id, node_name, service_name) (last_over_time(%s[%ds]))", MetricName, int64(window.Seconds()))

	req, err := s.newRequest(ctx, http.MethodGet, "api/v1/query", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = url.Values{"query": {query}}.Encode()

	body, err := s.do(req)
	if err != nil {
		return nil, err
	}

	var res struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
			} `json:"result"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}
	if res.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", res.Error)
	}

	keys := make([]Key, 0, len(res.Data.Result))
	for _, r := range res.Data.Result {
		id, err := strconv.ParseInt(r.Metric["annotation_id"], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid annotation_id '%s' in query response: %w", r.Metric["annotation_id"], err)
		}

		keys = append(keys, Key{
			AnnotationID: id,
			NodeName:     r.Metric["node_name"],
			ServiceName:  r.Metric["service_name"],
		})
	}

	return keys, nil
}

// timeSeries returns the event as a series with labels sorted by name, as remote write requires.
func (e Event) timeSeries() prompb.TimeSeries {
	labels := e.SeriesLabels()

	ts := prompb.TimeSeries{Labels: make([]prompb.Label, 0, len(labels))}
	for _, name := range slices.Sorted(maps.Keys(labels)) {
		ts.Labels = append(ts.Labels, prompb.Label{Name: name, Value: labels[name]})
	}

	ms := e.Time.UnixMilli()
	ts.Samples = []prompb.Sample{{Value: float64(ms), Timestamp: ms}}

	return ts
}

// newRequest returns a request to a path relative to the base URL, with basic auth if the base URL has userinfo.
func (s *Store) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.base.JoinPath(path).String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if s.user != nil {
		password, _ := s.user.Password()
		req.SetBasicAuth(s.user.Username(), password)
	}

	return req, nil
}

// do sends the request and returns the response body, or an error for a non-2xx status.
func (s *Store) do(req *http.Request) ([]byte, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected response status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return body, nil
}
