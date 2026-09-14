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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"

	actionsClient "github.com/percona/pmm/api/actions/v1/json/client"
	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	"github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
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

	// Bound of an error response body kept for the message.
	maxErrorBody = 4 << 10

	// A Prometheus vector sample is [<unix timestamp>, "<value>"].
	vectorSampleLen = 2
)

// client implements pmmAPI over the generated go-swagger clients (PMM's /v1
// API) and net/http (Grafana paths), all through the nginx loopback.
type client struct {
	base *url.URL
	http *http.Client

	server    *serverClient.PMMServerAPI
	inventory *inventoryClient.PMMInventoryAPI
	qan       *qanClient.PMMQANAPI
	actions   *actionsClient.PMMActionsAPI
}

// newClient builds a pmmAPI implementation against the given base URL.
func newClient(baseURL string, l *logrus.Entry) (*client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid loopback URL '%s': %w", baseURL, err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid loopback URL '%s': scheme and host are required", baseURL)
	}
	if base.Path == "" {
		base.Path = "/"
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unexpected default transport %T", http.DefaultTransport)
	}
	rt := &statusTransport{next: transport.Clone()}

	swagger := httptransport.New(base.Host, base.Path, []string{base.Scheme})
	swagger.Transport = rt
	swagger.SetLogger(l.WithField("subcomponent", "loopback"))

	return &client{
		base:      base,
		http:      &http.Client{Transport: rt},
		server:    serverClient.New(swagger, nil),
		inventory: inventoryClient.New(swagger, nil),
		qan:       qanClient.New(swagger, nil),
		actions:   actionsClient.New(swagger, nil),
	}, nil
}

// AuthenticateRequest implements runtime.ClientAuthInfoWriter: the caller's
// Authorization and Cookie headers are copied verbatim onto the backing call.
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

func (c *client) ListServices(ctx context.Context, auth callerAuth) (*services_service.ListServicesOKBody, error) {
	params := services_service.NewListServicesParamsWithContext(ctx).WithTimeout(callTimeout)
	res, err := c.inventory.ServicesService.ListServices(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) ListNodes(ctx context.Context, auth callerAuth) (*nodes_service.ListNodesOKBody, error) {
	params := nodes_service.NewListNodesParamsWithContext(ctx).WithTimeout(callTimeout)
	res, err := c.inventory.NodesService.ListNodes(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) GetReport(ctx context.Context, auth callerAuth, body qan_service.GetReportBody) (*qan_service.GetReportOKBody, error) {
	params := qan_service.NewGetReportParamsWithContext(ctx).WithTimeout(callTimeout).WithBody(body)
	res, err := c.qan.QANService.GetReport(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) GetMetrics(ctx context.Context, auth callerAuth, body qan_service.GetMetricsBody) (*qan_service.GetMetricsOKBody, error) {
	params := qan_service.NewGetMetricsParamsWithContext(ctx).WithTimeout(callTimeout).WithBody(body)
	res, err := c.qan.QANService.GetMetrics(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) GetQueryExample(ctx context.Context, auth callerAuth, body qan_service.GetQueryExampleBody) (*qan_service.GetQueryExampleOKBody, error) {
	params := qan_service.NewGetQueryExampleParamsWithContext(ctx).WithTimeout(callTimeout).WithBody(body)
	res, err := c.qan.QANService.GetQueryExample(params, auth.option())
	if err != nil {
		return nil, err
	}
	return res.Payload, nil
}

func (c *client) GetQueryPlan(ctx context.Context, auth callerAuth, queryID string) (*qan_service.GetQueryPlanOKBody, error) {
	params := qan_service.NewGetQueryPlanParamsWithContext(ctx).WithTimeout(callTimeout).WithQueryid(queryID)
	res, err := c.qan.QANService.GetQueryPlan(params, auth.option())
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

// ListDatasources returns Grafana's datasources; the tools look for the
// Prometheus-typed "Metrics" one (PMM's VictoriaMetrics behind vmproxy).
func (c *client) ListDatasources(ctx context.Context, auth callerAuth) ([]datasource, error) {
	var out []datasource
	err := c.getJSON(ctx, auth, "graph/api/datasources", nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryInstant runs an instant PromQL query through the Grafana datasource
// proxy (uid-based path), which is Viewer-accessible and LBAC-filtered, unlike
// the raw /prometheus path that requires admin.
func (c *client) QueryInstant(ctx context.Context, auth callerAuth, datasourceUID, promql string, at time.Time) ([]metricSample, error) {
	query := url.Values{"query": {promql}}
	if !at.IsZero() {
		query.Set("time", strconv.FormatInt(at.Unix(), 10))
	}
	path := "graph/api/datasources/proxy/uid/" + url.PathEscape(datasourceUID) + "/api/v1/query"

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	err := c.getJSON(ctx, auth, path, query, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, newToolError(codePMMUnavailable, "metrics query failed with status '%s'", resp.Status)
	}

	samples := make([]metricSample, 0, len(resp.Data.Result))
	for _, r := range resp.Data.Result {
		if len(r.Value) != vectorSampleLen {
			continue
		}
		var raw string
		err := json.Unmarshal(r.Value[1], &raw)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		samples = append(samples, metricSample{Labels: r.Metric, Value: v})
	}
	return samples, nil
}

// getJSON performs an authenticated GET on a path relative to the base URL and
// decodes the JSON response.
func (c *client) getJSON(ctx context.Context, auth callerAuth, path string, query url.Values, out any) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	u := c.base.ResolveReference(&url.URL{Path: path})
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	auth.apply(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return fmt.Errorf("decoding response of GET %s: %w", path, err)
	}
	return nil
}
