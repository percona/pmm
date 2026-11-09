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
	"net/http"
	"net/url"
	"os"
	"regexp"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	actionsClient "github.com/percona/pmm/api/actions/v1/json/client"
	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	qanClient "github.com/percona/pmm/api/qan/v1/json/client"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
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

// errResponseTooLarge is returned when a datasource-proxy response is longer than maxVMResponseBytes.
var errResponseTooLarge = errors.New("response from the metrics datasource is larger than the 4 MiB limit")

// promRequest holds the arguments of a VictoriaMetrics query; zero ones are not sent.
type promRequest struct {
	Query      string
	Time       time.Time
	Start, End time.Time
	Step       time.Duration
	Timeout    time.Duration
}

func (r promRequest) values() url.Values {
	v := url.Values{}
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
	if r.Timeout > 0 {
		v.Set("timeout", r.Timeout.String())
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

// client implements pmmAPI over the generated go-swagger clients (PMM's /v1
// API) and net/http (Grafana paths), all through the nginx loopback.
type client struct {
	base *url.URL
	http *http.Client

	swagger *httptransport.Runtime
	server  *serverClient.PMMServerAPI
	qan     *qanClient.PMMQANAPI
	actions *actionsClient.PMMActionsAPI

	vmSlots chan struct{}
}

// vmDefaultMaxConcurrentRequests is VictoriaMetrics' default -search.maxConcurrentRequests on cpus CPUs, as
// getDefaultMaxConcurrentRequests in app/victoria-metrics/main.go computes it at v1.154.0.
func vmDefaultMaxConcurrentRequests(cpus int) int {
	return min(vmRequestsPerCPU*cpus, maxVMDefaultRequests)
}

// maxVMRequestsPerReplica is half of VictoriaMetrics' -search.maxConcurrentRequests, so that
// dashboards and alert evaluation keep the other half.
func maxVMRequestsPerReplica() int {
	// VictoriaMetrics counts the CPUs as GOMAXPROCS, which it sets from the container's CPU quota.
	limit := vmDefaultMaxConcurrentRequests(goruntime.GOMAXPROCS(0))
	v, ok := os.LookupEnv("VM_search_maxConcurrentRequests")
	if ok && v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			limit = n
		}
	}
	return max(limit/vmReplicaShare, 1)
}

// newClient builds a pmmAPI implementation against the given base URL.
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
	if base.Path == "" {
		base.Path = "/"
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unexpected default transport %T", http.DefaultTransport)
	}
	rt := &statusTransport{next: transport.Clone()}

	// No SetLogger here: it assigns go-openapi's package-global
	// middleware.Logger, so it would re-route logging for every go-openapi
	// client in pmm-managed, and it races when clients are built concurrently.
	swagger := httptransport.New(base.Host, base.Path, []string{base.Scheme})
	swagger.Transport = rt
	// SWAGGER_DEBUG and DEBUG make go-openapi dump whole requests, Authorization and cookies included. Set the field,
	// not SetDebug: that writes a package global, which races between clients.
	swagger.Debug = false

	return &client{
		base:    base,
		http:    &http.Client{Transport: rt},
		swagger: swagger,
		server:  serverClient.New(swagger, nil),
		qan:     qanClient.New(swagger, nil),
		actions: actionsClient.New(swagger, nil),
		vmSlots: make(chan struct{}, maxVMRequestsPerReplica()),
	}, nil
}

// AuthenticateRequest implements runtime.ClientAuthInfoWriter: the caller's
// credentials are set on the backing call as callerAuthFromHeader canonicalized them.
func (a callerAuth) AuthenticateRequest(r runtime.ClientRequest, _ strfmt.Registry) error {
	if a.authorization != "" {
		err := r.SetHeaderParam("Authorization", a.authorization)
		if err != nil {
			return err
		}
	}
	if a.cookie != "" {
		err := r.SetHeaderParam("Cookie", a.cookie)
		if err != nil {
			return err
		}
	}
	return nil
}

// apply sets the caller's headers on a raw net/http request.
func (a callerAuth) apply(req *http.Request) {
	if a.authorization != "" {
		req.Header.Set("Authorization", a.authorization)
	}
	if a.cookie != "" {
		req.Header.Set("Cookie", a.cookie)
	}
}

// option returns a generated-client option that authenticates one operation
// with the caller's credentials. The generated packages each declare their own
// ClientOption type over the same function signature.
func (a callerAuth) option() func(*runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		op.AuthInfo = a
	}
}

// withQueryParam adds a query parameter the generated client does not declare.
// A parameter the request message lacks is ignored by grpc-gateway, so it can
// be sent before the server supports it.
func withQueryParam(name, value string) func(*runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		params := op.Params
		op.Params = runtime.ClientRequestWriterFunc(func(r runtime.ClientRequest, reg strfmt.Registry) error {
			err := params.WriteToRequest(r, reg)
			if err != nil {
				return err
			}
			return r.SetQueryParam(name, value)
		})
	}
}

// statusTransport turns every non-2xx response into a statusError so that the
// generated clients and the raw Grafana calls share one error mapping.
type statusTransport struct {
	next http.RoundTripper
}

func (t *statusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
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
	params := server_service.NewVersionParamsWithContext(ctx).WithTimeout(callTimeout)
	res, err := c.server.ServerService.Version(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

// GetReport and GetMetrics bypass the generated client: qan-api2 encodes NaN
// sparkline values as strings, which the swagger types reject (see lenientFloat).
func (c *client) GetReport(ctx context.Context, auth callerAuth, body qan_service.GetReportBody) (*qanReport, error) {
	var out qanReport
	err := c.postJSON(ctx, auth, "v1/qan/metrics:getReport", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *client) GetMetrics(ctx context.Context, auth callerAuth, body qan_service.GetMetricsBody) (*queryMetrics, error) {
	var out queryMetrics
	err := c.postJSON(ctx, auth, "v1/qan:getMetrics", body, &out)
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
	err := c.postJSON(ctx, auth, "v1/qan/metrics:getFilters", body, &out)
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
	params := qan_service.NewGetQueryExampleParamsWithContext(ctx).WithTimeout(callTimeout).WithBody(body)
	res, err := c.qan.QANService.GetQueryExample(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) GetQueryPlan(ctx context.Context, auth callerAuth, queryID, serviceID string) (*qan_service.GetQueryPlanOKBody, error) {
	params := qan_service.NewGetQueryPlanParamsWithContext(ctx).WithTimeout(callTimeout).WithQueryid(queryID)
	res, err := c.qan.QANService.GetQueryPlan(params, auth.option(), withQueryParam("service_id", serviceID))
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) StartServiceAction(
	ctx context.Context, auth callerAuth, body actions_service.StartServiceActionBody,
) (*actions_service.StartServiceActionOKBody, error) {
	params := actions_service.NewStartServiceActionParamsWithContext(ctx).WithTimeout(callTimeout).WithBody(body)
	res, err := c.actions.ActionsService.StartServiceAction(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) GetAction(ctx context.Context, auth callerAuth, actionID string) (*actions_service.GetActionOKBody, error) {
	params := actions_service.NewGetActionParamsWithContext(ctx).WithTimeout(callTimeout).WithActionID(actionID)
	res, err := c.actions.ActionsService.GetAction(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) CancelAction(ctx context.Context, auth callerAuth, actionID string) error {
	params := actions_service.NewCancelActionParamsWithContext(ctx).WithTimeout(callTimeout).
		WithBody(actions_service.CancelActionBody{ActionID: actionID})
	_, err := c.actions.ActionsService.CancelAction(params, auth.option())
	return err
}

// GetDatasourceByName returns the Grafana datasource with the given name.
func (c *client) GetDatasourceByName(ctx context.Context, auth callerAuth, name string) (*datasource, error) {
	var out datasource
	err := c.getJSON(ctx, auth, "graph/api/datasources/name/"+url.PathEscape(name), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// QueryInstant runs an instant PromQL query through the Grafana datasource
// proxy (uid-based path), which is Viewer-accessible and LBAC-filtered, unlike
// the raw /prometheus path that requires admin.
func (c *client) QueryInstant(ctx context.Context, auth callerAuth, uid string, r promRequest) ([]metricSample, error) {
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	err := c.getVM(ctx, auth, uid, "api/v1/query", r.values(), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, newToolError(codePMMUnavailable, "metrics query failed with status '%s'", resp.Status)
	}

	samples := make([]metricSample, 0, len(resp.Data.Result))
	for _, res := range resp.Data.Result {
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
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string   `json:"metric"`
				Values [][]json.RawMessage `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	err := c.getVM(ctx, auth, uid, "api/v1/query_range", r.values(), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, newToolError(codePMMUnavailable, "metrics query failed with status '%s'", resp.Status)
	}

	out := make([]metricSeries, 0, len(resp.Data.Result))
	for _, res := range resp.Data.Result {
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
	return c.listStrings(ctx, auth, uid, "api/v1/label/"+url.PathEscape(label)+"/values", match, start, end, limit)
}

// Labels lists the label names of the series matching match.
func (c *client) Labels(ctx context.Context, auth callerAuth, uid string, match []string, start, end time.Time, limit int) ([]string, error) {
	return c.listStrings(ctx, auth, uid, "api/v1/labels", match, start, end, limit)
}

// Series lists the label sets of the series matching match.
func (c *client) Series(ctx context.Context, auth callerAuth, uid string, match []string, start, end time.Time, limit int) ([]map[string]string, error) {
	var resp struct {
		Status string              `json:"status"`
		Data   []map[string]string `json:"data"`
	}
	err := c.getVM(ctx, auth, uid, "api/v1/series", listValues(match, start, end, limit), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, newToolError(codePMMUnavailable, "metrics series request failed with status '%s'", resp.Status)
	}
	return resp.Data, nil
}

func (c *client) listStrings(ctx context.Context, auth callerAuth, uid, endpoint string, match []string, start, end time.Time, limit int) ([]string, error) {
	var resp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	err := c.getVM(ctx, auth, uid, endpoint, listValues(match, start, end, limit), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, newToolError(codePMMUnavailable, "metrics list request failed with status '%s'", resp.Status)
	}
	return resp.Data, nil
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

// getVM sends one request to VictoriaMetrics through the datasource proxy, holding one of the replica's
// VictoriaMetrics slots until it has read the response, of which it reads at most maxVMResponseBytes.
func (c *client) getVM(ctx context.Context, auth callerAuth, uid, endpoint string, query url.Values, out any) error {
	select {
	case c.vmSlots <- struct{}{}:
		defer func() { <-c.vmSlots }()
	case <-ctx.Done():
		return newToolError(codeTimeout, "no free metrics request slot before the deadline: %s", ctx.Err())
	}

	path := "graph/api/datasources/proxy/uid/" + url.PathEscape(uid) + "/" + endpoint
	return c.doJSONLimited(ctx, auth, http.MethodGet, path, query, nil, out, maxVMResponseBytes)
}

// getJSON performs an authenticated GET on a path relative to the base URL and
// decodes the JSON response.
func (c *client) getJSON(ctx context.Context, auth callerAuth, path string, query url.Values, out any) error {
	return c.doJSON(ctx, auth, http.MethodGet, path, query, nil, out)
}

// postJSON performs an authenticated POST with a JSON body and decodes the response.
func (c *client) postJSON(ctx context.Context, auth callerAuth, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request for POST %s: %w", path, err)
	}
	return c.doJSON(ctx, auth, http.MethodPost, path, nil, b, out)
}

func (c *client) doJSON(ctx context.Context, auth callerAuth, method, path string, query url.Values, body []byte, out any) error {
	return c.doJSONLimited(ctx, auth, method, path, query, body, out, 0)
}

// doJSONLimited is doJSON that fails with errResponseTooLarge when the response is longer than
// maxBytes; zero means no limit.
func (c *client) doJSONLimited(ctx context.Context, auth callerAuth, method, path string, query url.Values, body []byte, out any, maxBytes int64) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	u := c.base.ResolveReference(&url.URL{Path: path})
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	auth.apply(req)

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
			return errResponseTooLarge
		}
		src = bytes.NewReader(b)
	}

	err = json.NewDecoder(src).Decode(out)
	if err != nil {
		return fmt.Errorf("decoding response of %s %s: %w", method, path, err)
	}
	return nil
}
