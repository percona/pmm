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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

const (
	// DefaultLoopbackURL is nginx's plain-HTTP listener inside the PMM Server
	// container. Every backing call re-enters nginx here so that auth_request
	// authorizes it against the caller's role exactly like an external request.
	DefaultLoopbackURL = "http://127.0.0.1:8080/"

	// Bound of a single PMM API request.
	callTimeout = 30 * time.Second

	// Bound of a VictoriaMetrics query, below callTimeout so that VictoriaMetrics answers first.
	metricsTimeout = 25 * time.Second

	// Bound of any datasource-proxy response read into memory.
	maxVMResponseBytes = 4 << 20

	// Bound of any response of Grafana's alerting APIs read into memory.
	maxAlertingResponseBytes = 16 << 20

	// Bound of QAN's filter lists read into memory.
	maxQANFiltersBytes = 4 << 20

	// Bound of a QAN report or query-metrics response read into memory. The largest one MCP asks for, a 200-row
	// search report with 14 columns, 239-point sparklines and 2,048-character fingerprints, is about 9.2 MiB.
	maxQANResponseBytes = 16 << 20

	// One mebibyte, the unit in which a read limit is reported.
	mib = 1 << 20

	// VictoriaMetrics' default -search.maxConcurrentRequests is vmRequestsPerCPU per CPU, at most maxVMDefaultRequests.
	vmRequestsPerCPU     = 2
	maxVMDefaultRequests = 16

	// VictoriaMetrics' slots MCP may use: one in vmReplicaShare.
	vmReplicaShare = 2

	// Bound of an error response body kept for the message.
	maxErrorBody = 4 << 10

	// A Prometheus vector sample is [<unix timestamp>, "<value>"].
	vectorSampleLen = 2
)

// errResponseTooLarge is wrapped in the error of a response longer than the bound its call reads.
var errResponseTooLarge = errors.New("response is larger than the read limit")

// promRequest holds the arguments of a VictoriaMetrics query; zero ones are not sent, and the timeout argument is
// always metricsTimeout.
type promRequest struct {
	Query      string
	Time       time.Time
	Start, End time.Time
	Step       time.Duration
}

func (r promRequest) values() url.Values {
	v := url.Values{"timeout": {metricsTimeout.String()}}
	if r.Query != "" {
		v.Set("query", r.Query)
	}
	if !r.Time.IsZero() {
		v.Set("time", strconv.FormatInt(r.Time.Unix(), 10))
	}
	if !r.Start.IsZero() {
		v.Set("start", strconv.FormatInt(r.Start.Unix(), 10))
	}
	if !r.End.IsZero() {
		v.Set("end", strconv.FormatInt(r.End.Unix(), 10))
	}
	if r.Step > 0 {
		v.Set("step", strconv.FormatInt(int64(r.Step/time.Second), 10))
	}
	return v
}

// point is one sample of a range query.
type point struct {
	T time.Time
	V float64
}

// metricSeries is one series of a range query.
type metricSeries struct {
	Labels map[string]string
	Points []point
}

// client implements pmmAPI with net/http, every call through the nginx loopback.
type client struct {
	base    *url.URL
	http    *http.Client
	vmSlots chan struct{}
}

// vmDefaultMaxConcurrentRequests is VictoriaMetrics' default -search.maxConcurrentRequests on cpus CPUs, as
// getDefaultMaxConcurrentRequests in app/victoria-metrics/main.go computes it at v1.156.0.
func vmDefaultMaxConcurrentRequests(cpus int) int {
	return min(vmRequestsPerCPU*cpus, maxVMDefaultRequests)
}

// maxVMRequestsPerReplica is half of VictoriaMetrics' -search.maxConcurrentRequests, so that
// dashboards and alert evaluation keep the other half.
func maxVMRequestsPerReplica() int {
	// VictoriaMetrics counts the CPUs as GOMAXPROCS, which it sets from the container's CPU quota.
	limit := vmDefaultMaxConcurrentRequests(runtime.GOMAXPROCS(0))
	v, ok := os.LookupEnv("VM_search_maxConcurrentRequests")
	if ok && v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			limit = n
		}
	}
	return max(limit/vmReplicaShare, 1)
}

// newClient builds a pmmAPI implementation against the given base URL, whose host must be a loopback address or
// localhost, so that the caller's credentials never leave the server.
//
// Its errors never echo the URL: it comes from PMM_DEV_MCP_LOOPBACK_URL, can
// carry userinfo or a query token, and main logs this error on startup.
func newClient(baseURL string) (*client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.New("invalid loopback URL (PMM_DEV_MCP_LOOPBACK_URL): it does not parse as a URL")
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, errors.New("invalid loopback URL (PMM_DEV_MCP_LOOPBACK_URL): scheme and host are required")
	}
	host := base.Hostname()
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return nil, errors.New("invalid loopback URL (PMM_DEV_MCP_LOOPBACK_URL): the host must be a loopback address or localhost")
	}
	if base.Path == "" {
		base.Path = "/"
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unexpected default transport %T", http.DefaultTransport)
	}

	return &client{
		base:    base,
		http:    &http.Client{Transport: &statusTransport{next: transport.Clone()}},
		vmSlots: make(chan struct{}, maxVMRequestsPerReplica()),
	}, nil
}

// statusTransport turns every non-2xx response into a statusError, so that every backing call shares one error
// mapping and none follows a redirect.
type statusTransport struct {
	next http.RoundTripper
}

func (t *statusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		// An error here stops the HTTP client from following the redirect, so the caller's credentials stay on the loopback.
		_ = resp.Body.Close()
		return nil, &statusError{status: resp.StatusCode, method: req.Method, path: req.URL.Path, message: "a backing call does not follow redirects"}
	}
	if resp.StatusCode < http.StatusBadRequest {
		return resp, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	_ = resp.Body.Close()

	return nil, &statusError{
		status:  resp.StatusCode,
		method:  req.Method,
		path:    req.URL.Path,
		message: messageFromBody(body),
	}
}

// messageFromBody extracts PMM's JSON error message ({"code":7,"error":"…","message":"…"})
// and falls back to the trimmed body.
func messageFromBody(body []byte) string {
	var m struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	err := json.Unmarshal(body, &m)
	if err == nil {
		if m.Message != "" {
			return m.Message
		}
		if m.Error != "" {
			return m.Error
		}
	}
	return strings.TrimSpace(string(body))
}

func (c *client) Version(ctx context.Context, auth callerAuth) (*server_service.VersionOKBody, error) {
	var out server_service.VersionOKBody
	err := c.do(ctx, auth, http.MethodGet, "v1/server/version", nil, nil, &out, 0)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetReport and GetMetrics decode into their own types, not the generated ones: qan-api2 encodes NaN sparkline values
// as strings, which the generated types reject (see lenientFloat).
func (c *client) GetReport(ctx context.Context, auth callerAuth, body qan_service.GetReportBody) (*qanReport, error) {
	var out qanReport
	err := c.do(ctx, auth, http.MethodPost, "v1/qan/metrics:getReport", nil, body, &out, maxQANResponseBytes)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *client) GetMetrics(ctx context.Context, auth callerAuth, body qan_service.GetMetricsBody) (*queryMetrics, error) {
	var out queryMetrics
	err := c.do(ctx, auth, http.MethodPost, "v1/qan:getMetrics", nil, body, &out, maxQANResponseBytes)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// QANServiceTypes returns the service types QAN holds data for in a window.
func (c *client) QANServiceTypes(ctx context.Context, auth callerAuth, from, to time.Time) ([]string, error) {
	var out struct {
		Labels map[string]struct {
			Name []struct {
				Value string `json:"value"`
			} `json:"name"`
		} `json:"labels"`
	}
	body := qan_service.GetFilteredMetricsNamesBody{
		PeriodStartFrom: strfmt.DateTime(from),
		PeriodStartTo:   strfmt.DateTime(to),
		Labels:          []*qan_service.GetFilteredMetricsNamesParamsBodyLabelsItems0{},
	}
	err := c.do(ctx, auth, http.MethodPost, "v1/qan/metrics:getFilters", nil, body, &out, maxQANFiltersBytes)
	if err != nil {
		return nil, err
	}
	types := make([]string, 0, len(out.Labels["service_type"].Name))
	for _, v := range out.Labels["service_type"].Name {
		types = append(types, v.Value)
	}
	return types, nil
}

func (c *client) GetQueryExample(ctx context.Context, auth callerAuth, body qan_service.GetQueryExampleBody) (*qan_service.GetQueryExampleOKBody, error) {
	var out qan_service.GetQueryExampleOKBody
	err := c.do(ctx, auth, http.MethodPost, "v1/qan/query:getExample", nil, body, &out, 0)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *client) GetQueryPlan(ctx context.Context, auth callerAuth, queryID, serviceID string) (*qan_service.GetQueryPlanOKBody, error) {
	var query url.Values
	if serviceID != "" {
		query = url.Values{"service_id": {serviceID}}
	}
	var out qan_service.GetQueryPlanOKBody
	err := c.do(ctx, auth, http.MethodGet, "v1/qan/query/"+url.PathEscape(queryID)+"/plan", query, nil, &out, 0)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *client) StartServiceAction(
	ctx context.Context, auth callerAuth, body actions_service.StartServiceActionBody,
) (*actions_service.StartServiceActionOKBody, error) {
	var out actions_service.StartServiceActionOKBody
	err := c.do(ctx, auth, http.MethodPost, "v1/actions:startServiceAction", nil, body, &out, 0)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *client) GetAction(ctx context.Context, auth callerAuth, actionID string) (*actions_service.GetActionOKBody, error) {
	var out actions_service.GetActionOKBody
	err := c.do(ctx, auth, http.MethodGet, "v1/actions/"+url.PathEscape(actionID), nil, nil, &out, 0)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *client) CancelAction(ctx context.Context, auth callerAuth, actionID string) error {
	var out any
	return c.do(ctx, auth, http.MethodPost, "v1/actions:cancelAction", nil, actions_service.CancelActionBody{ActionID: actionID}, &out, 0)
}

// GetDatasourceByName returns the Grafana datasource with the given name.
func (c *client) GetDatasourceByName(ctx context.Context, auth callerAuth, name string) (*datasource, error) {
	var out datasource
	err := c.do(ctx, auth, http.MethodGet, "graph/api/datasources/name/"+url.PathEscape(name), nil, nil, &out, 0)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// QueryInstant runs an instant PromQL query at step instantStep through the Grafana datasource
// proxy (uid-based path), which is Viewer-accessible and LBAC-filtered, unlike
// the raw /prometheus path that requires admin.
func (c *client) QueryInstant(ctx context.Context, auth callerAuth, uid string, r promRequest) ([]metricSample, error) {
	type vector struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []json.RawMessage `json:"value"`
		} `json:"result"`
	}
	// Without a step, VictoriaMetrics resolves i units and implicit subquery steps at its lookback delta, which its
	// flags can set apart from instantStep, the step the checks measure an instant query at.
	r.Step = instantStep
	data, err := promData[vector](ctx, c, auth, uid, "api/v1/query", r.values(), "metrics query")
	if err != nil {
		return nil, err
	}

	samples := make([]metricSample, 0, len(data.Result))
	for _, res := range data.Result {
		if len(res.Value) != vectorSampleLen {
			continue
		}
		v, ok := parseSampleValue(res.Value[1])
		if !ok {
			continue
		}
		samples = append(samples, metricSample{Labels: res.Metric, Value: v})
	}
	return samples, nil
}

// QueryRange runs a range PromQL query through the Grafana datasource proxy.
func (c *client) QueryRange(ctx context.Context, auth callerAuth, uid string, r promRequest) ([]metricSeries, error) {
	type matrix struct {
		Result []struct {
			Metric map[string]string   `json:"metric"`
			Values [][]json.RawMessage `json:"values"`
		} `json:"result"`
	}
	data, err := promData[matrix](ctx, c, auth, uid, "api/v1/query_range", r.values(), "metrics query")
	if err != nil {
		return nil, err
	}

	out := make([]metricSeries, 0, len(data.Result))
	for _, res := range data.Result {
		points := make([]point, 0, len(res.Values))
		for _, val := range res.Values {
			if len(val) != vectorSampleLen {
				continue
			}
			var ts float64
			err := json.Unmarshal(val[0], &ts)
			if err != nil {
				continue
			}
			v, ok := parseSampleValue(val[1])
			if !ok {
				continue
			}
			sec, frac := math.Modf(ts)
			points = append(points, point{T: time.Unix(int64(sec), int64(frac*float64(time.Second))).UTC(), V: v})
		}
		out = append(out, metricSeries{Labels: res.Metric, Points: points})
	}
	return out, nil
}

// labelNamePattern is what a label name may look like; url.PathEscape leaves a name such as '..' alone, so the path segment is checked.
var labelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func errBadLabel(label string) error {
	return newToolError(codeInvalidInput, "label '%s' is not a label name: use letters, digits and underscores, not starting with a digit", echo(label))
}

// LabelValues lists the values of a label of the series matching match.
func (c *client) LabelValues(ctx context.Context, auth callerAuth, uid, label string, match []string, start, end time.Time, limit int) ([]string, error) {
	if !labelNamePattern.MatchString(label) {
		return nil, errBadLabel(label)
	}
	endpoint := "api/v1/label/" + url.PathEscape(label) + "/values"
	return promData[[]string](ctx, c, auth, uid, endpoint, listValues(match, start, end, limit), "metrics list request")
}

// Labels lists the label names of the series matching match.
func (c *client) Labels(ctx context.Context, auth callerAuth, uid string, match []string, start, end time.Time, limit int) ([]string, error) {
	return promData[[]string](ctx, c, auth, uid, "api/v1/labels", listValues(match, start, end, limit), "metrics list request")
}

// Series lists the label sets of the series matching match.
func (c *client) Series(ctx context.Context, auth callerAuth, uid string, match []string, start, end time.Time, limit int) ([]map[string]string, error) {
	return promData[[]map[string]string](ctx, c, auth, uid, "api/v1/series", listValues(match, start, end, limit), "metrics series request")
}

// listValues is the argument set of the label, label-values and series endpoints.
func listValues(match []string, start, end time.Time, limit int) url.Values {
	v := url.Values{}
	if len(match) > 0 {
		v["match[]"] = match
	}
	if !start.IsZero() {
		v.Set("start", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		v.Set("end", strconv.FormatInt(end.Unix(), 10))
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	return v
}

// Alerts returns the firing, recovering and pending alerts of the rules Grafana's rules API returns to the caller, which follows the caller's folder permissions.
func (c *client) Alerts(ctx context.Context, auth callerAuth) ([]grafanaAlert, error) {
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Groups []struct {
				Rules []struct {
					UID    string `json:"uid"`
					Alerts []struct {
						Labels      map[string]string `json:"labels"`
						Annotations map[string]string `json:"annotations"`
						State       string            `json:"state"`
						ActiveAt    *time.Time        `json:"activeAt"`
					} `json:"alerts"`
				} `json:"rules"`
			} `json:"groups"`
		} `json:"data"`
	}
	err := c.getAlerting(ctx, auth, "graph/api/prometheus/grafana/api/v1/rules", url.Values{"state": {"firing", "recovering", "pending"}}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, newToolError(codePMMUnavailable, "alert rules request failed with status '%s'", resp.Status)
	}

	var out []grafanaAlert
	for _, group := range resp.Data.Groups {
		for _, rule := range group.Rules {
			for _, a := range rule.Alerts {
				// A state is Alerting, Recovering or Pending, alone or with a reason, such as "Alerting (NoData)".
				firing := a.State == "Alerting" || strings.HasPrefix(a.State, "Alerting (")
				recovering := a.State == "Recovering" || strings.HasPrefix(a.State, "Recovering (")
				pending := a.State == "Pending" || strings.HasPrefix(a.State, "Pending (")
				if !firing && !recovering && !pending {
					continue
				}
				alert := grafanaAlert{RuleUID: rule.UID, Labels: a.Labels, Annotations: a.Annotations, Pending: pending, Recovering: recovering}
				if a.ActiveAt != nil {
					alert.StartsAt = a.ActiveAt.UTC()
				}
				out = append(out, alert)
			}
		}
	}
	return out, nil
}

// SilencedAlerts returns the Alertmanager's alerts that are not active: silenced, inhibited or not yet processed.
func (c *client) SilencedAlerts(ctx context.Context, auth callerAuth) ([]amAlert, error) {
	var out []amAlert
	err := c.getAlerting(ctx, auth, "graph/api/alertmanager/grafana/api/v2/alerts", url.Values{"active": {"false"}}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Silences returns the Alertmanager's silences.
func (c *client) Silences(ctx context.Context, auth callerAuth) ([]amSilence, error) {
	var out []amSilence
	err := c.getAlerting(ctx, auth, "graph/api/alertmanager/grafana/api/v2/silences", nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// parseSampleValue reads a Prometheus sample value, a JSON string that may be NaN or Inf.
func parseSampleValue(raw json.RawMessage) (float64, bool) {
	var s string
	err := json.Unmarshal(raw, &s)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// promData sends one request to VictoriaMetrics through the datasource proxy and returns the data of its answer, or
// pmm_unavailable naming the request as what when the answer's status is not success. It holds one of the replica's
// VictoriaMetrics slots until it has read the response, of which it reads at most maxVMResponseBytes.
func promData[T any](ctx context.Context, c *client, auth callerAuth, uid, endpoint string, query url.Values, what string) (T, error) { //nolint:ireturn
	var zero T
	select {
	case c.vmSlots <- struct{}{}:
		defer func() { <-c.vmSlots }()
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			return zero, fmt.Errorf("waiting for a metrics request slot: %w", err)
		}
		return zero, newToolError(codeTimeout, "no free metrics request slot before the deadline: %s", err)
	}

	var resp struct {
		Status string `json:"status"`
		Data   T      `json:"data"`
	}
	path := "graph/api/datasources/proxy/uid/" + url.PathEscape(uid) + "/" + endpoint
	err := c.do(ctx, auth, http.MethodGet, path, query, nil, &resp, maxVMResponseBytes)
	if err != nil {
		return zero, err
	}
	if resp.Status != "success" {
		return zero, newToolError(codePMMUnavailable, "%s failed with status '%s'", what, resp.Status)
	}
	return resp.Data, nil
}

// getAlerting sends a GET to one of Grafana's alerting APIs, of whose response it reads at most maxAlertingResponseBytes.
func (c *client) getAlerting(ctx context.Context, auth callerAuth, path string, query url.Values, out any) error {
	err := c.do(ctx, auth, http.MethodGet, path, query, nil, out, maxAlertingResponseBytes)
	if errors.Is(err, errResponseTooLarge) {
		return newToolError(codePMMUnavailable,
			"Grafana's response to GET /%s is larger than the %d MiB limit; open PMM's Alerting page to see the alerts", path, maxAlertingResponseBytes/mib)
	}
	return err
}

// do sends one request with the caller's credentials to path, escaped and relative to the loopback URL, with body
// encoded as JSON unless it is nil, and decodes the JSON response into out. It reads at most maxBytes of the response,
// all of it when maxBytes is zero, and fails with an error wrapping errResponseTooLarge on a longer one.
func (c *client) do(ctx context.Context, auth callerAuth, method, path string, query url.Values, body, out any, maxBytes int64) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	u, err := c.base.Parse(path)
	if err != nil {
		return err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request for %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth.authorization != "" {
		req.Header.Set("Authorization", auth.authorization)
	}
	if auth.cookie != "" {
		req.Header.Set("Cookie", auth.cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	var src io.Reader = resp.Body
	if maxBytes > 0 {
		b, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
		if err != nil {
			return fmt.Errorf("reading response of %s %s: %w", method, path, err)
		}
		if int64(len(b)) > maxBytes {
			return fmt.Errorf("%s /%s: %w of %d MiB", method, path, errResponseTooLarge, maxBytes/mib)
		}
		src = bytes.NewReader(b)
	}

	err = json.NewDecoder(src).Decode(out)
	if err != nil {
		return fmt.Errorf("decoding response of %s %s: %w", method, path, err)
	}
	return nil
}
