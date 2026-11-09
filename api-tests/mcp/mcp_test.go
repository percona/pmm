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

// Package mcp contains integration tests for PMM's MCP endpoint (/mcp).
package mcp

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
	"github.com/percona/pmm/utils/tlsconfig"
)

// basicAuthTransport adds the credentials from the test server URL to every request.
type basicAuthTransport struct {
	next     http.RoundTripper
	user     string
	password string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.SetBasicAuth(t.user, t.password)
	return t.next.RoundTrip(req)
}

// authorizationTransport sends the Authorization value on every request.
type authorizationTransport struct {
	next          http.RoundTripper
	authorization string
}

func (t *authorizationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", t.authorization)
	return t.next.RoundTrip(req)
}

// httpClient returns an HTTP client for the PMM Server under test; withAuth
// adds the admin credentials from the server URL.
func httpClient(withAuth bool) *http.Client {
	transport := &http.Transport{}
	if pmmapitests.BaseURL.Scheme == "https" {
		transport.TLSClientConfig = tlsconfig.Get()
		transport.TLSClientConfig.ServerName = pmmapitests.BaseURL.Hostname()
		transport.TLSClientConfig.InsecureSkipVerify = pmmapitests.ServerInsecureTLS
	}
	var rt http.RoundTripper = transport
	if withAuth && pmmapitests.BaseURL.User != nil {
		password, _ := pmmapitests.BaseURL.User.Password()
		rt = &basicAuthTransport{next: transport, user: pmmapitests.BaseURL.User.Username(), password: password}
	}
	return &http.Client{Transport: rt}
}

// endpoint returns the /mcp URL without the userinfo of the test server URL:
// net/http would otherwise turn it into basic auth on its own.
func endpoint() string {
	u := pmmapitests.BaseURL.ResolveReference(&url.URL{Path: "mcp"})
	u.User = nil
	return u.String()
}

// connect opens an authenticated MCP session against the server under test.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "pmm-api-tests", Version: "1"}, nil)
	session, err := client.Connect(pmmapitests.Context, &mcp.StreamableClientTransport{
		Endpoint:   endpoint(),
		HTTPClient: httpClient(true),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// connectWith opens an MCP session that sends authorization, a Bearer token or Basic credentials, on every request.
func connectWith(t *testing.T, authorization string) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "pmm-api-tests", Version: "1"}, nil)
	session, err := client.Connect(pmmapitests.Context, &mcp.StreamableClientTransport{
		Endpoint:   endpoint(),
		HTTPClient: &http.Client{Transport: &authorizationTransport{next: httpClient(false).Transport, authorization: authorization}},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

const (
	maxRateLimitedTries = 8
	errorMetaKey        = "com.percona/error"
)

// callRetrying calls a tool, sleeping retry_after (1 s when absent) and trying again while it answers rate_limited, up to maxRateLimitedTries calls.
func callRetrying(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	for range maxRateLimitedTries {
		res, err := session.CallTool(pmmapitests.Context, &mcp.CallToolParams{Name: name, Arguments: args})
		require.NoError(t, err)
		meta, _ := res.Meta[errorMetaKey].(map[string]any)
		if !res.IsError || meta["code"] != "rate_limited" {
			return res
		}
		wait := time.Second
		seconds, ok := meta["retry_after"].(float64)
		if ok && seconds >= 1 {
			wait = time.Duration(seconds) * time.Second
		}
		t.Logf("%s was rate limited; retrying in %s", name, wait)
		select {
		case <-pmmapitests.Context.Done():
			require.FailNow(t, "canceled while waiting to retry")
		case <-time.After(wait):
		}
	}
	require.FailNow(t, "still rate limited", "%s answered rate_limited %d times in a row", name, maxRateLimitedTries)
	return nil
}

// callTool calls a tool and returns its result and text content.
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()

	res := callRetrying(t, session, name, args)
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "unexpected content %T", res.Content[0])
	t.Logf("%s -> isError=%v\n%s", name, res.IsError, text.Text)
	return res, text.Text
}

// callText calls a tool and returns its text content and error flag.
func callText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()

	res, text := callTool(t, session, name, args)
	return text, res.IsError
}

// callResult calls a tool that must succeed, decodes its structured content into out and returns its text.
func callResult(t *testing.T, session *mcp.ClientSession, name string, args map[string]any, out any) string {
	t.Helper()

	res, text := callTool(t, session, name, args)
	require.False(t, res.IsError, text)
	b, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, out))
	return text
}

func TestUnauthenticated(t *testing.T) {
	t.Parallel()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"pmm-api-tests","version":"1"}}}`
	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodPost, endpoint(), strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := httpClient(false).Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Logf("Response: %s", b)

	// nginx auth_request answers 401 with pmm-managed's JSON body; the MCP
	// handler itself never emits 401.
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.EqualValues(t, 16, m["code"])
	assert.Equal(t, "Unauthorized", m["message"])
}

func TestListTools(t *testing.T) {
	t.Parallel()

	session := connect(t)
	assert.Equal(t, "pmm-mcp", session.InitializeResult().ServerInfo.Name)

	res, err := session.ListTools(pmmapitests.Context, nil)
	require.NoError(t, err)

	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		require.NotNil(t, tool.Annotations, tool.Name)
		assert.True(t, tool.Annotations.ReadOnlyHint, tool.Name)
		assert.NotEmpty(t, tool.OutputSchema, tool.Name)
	}
	for _, want := range []string{"pmm_version", "pmm_inventory"} {
		assert.Contains(t, names, want)
	}
}

func TestReadOnlySettingsReportMCP(t *testing.T) {
	t.Parallel()

	settings, err := serverClient.Default.ServerService.GetSettings(&server_service.GetSettingsParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	readOnly, err := serverClient.Default.ServerService.GetReadOnlySettings(&server_service.GetReadOnlySettingsParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	// TestMain switched the endpoint on, and the read-only view must agree
	// with the full settings. The shipped default, off, is pinned by
	// TestMCPDefaultsAreOff in managed/models rather than here, since this
	// package cannot know how the server under test was started.
	assert.True(t, settings.Payload.Settings.EnableMcp)
	assert.Equal(t, settings.Payload.Settings.EnableMcp, readOnly.Payload.Settings.EnableMcp)
}

func TestVersion(t *testing.T) {
	t.Parallel()

	text, isError := callText(t, connect(t), "pmm_version", nil)
	assert.False(t, isError)
	assert.True(t, strings.HasPrefix(text, "PMM Server 3."), text)
}

// topQueryID returns a service and one of its queryids from QAN, skipping the
// test when no service of the engine has QAN data yet. It tries every service,
// since the inventory can hold ones whose databases are gone.
func topQueryID(t *testing.T, session *mcp.ClientSession, engine string) (string, string, string) {
	t.Helper()

	res, err := inventoryClient.Default.ServicesService.ListServices(&services_service.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	type service struct{ id, name string }
	var candidates []service
	switch engine {
	case "mysql":
		for _, svc := range res.Payload.Mysql {
			candidates = append(candidates, service{svc.ServiceID, svc.ServiceName})
		}
	case "postgresql":
		for _, svc := range res.Payload.Postgresql {
			if svc.ServiceName == "pmm-server-postgresql" {
				continue
			}
			candidates = append(candidates, service{svc.ServiceID, svc.ServiceName})
		}
	}
	if len(candidates) == 0 {
		t.Skipf("no %s service registered", engine)
	}

	// MySQL queryids are hex digests, PostgreSQL ones signed decimals.
	re := regexp.MustCompile(`\[(-?[0-9A-Za-z]+)\]`)
	for _, svc := range candidates {
		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"service_name": svc.name, "period_from": "now-12h", "limit": 3})
		require.False(t, isError, text)
		m := re.FindStringSubmatch(text)
		if m != nil {
			return svc.id, svc.name, m[1]
		}
	}
	t.Skipf("no QAN data for any %s service yet", engine)
	return "", "", ""
}

func TestTopQueriesAndDetail(t *testing.T) {
	t.Parallel()

	session := connect(t)
	for _, engine := range []string{"mysql", "postgresql"} {
		t.Run(engine, func(t *testing.T) {
			t.Parallel()

			serviceID, serviceName, queryID := topQueryID(t, session, engine)

			text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": queryID, "service_id": serviceID, "period_from": "now-12h"})
			require.False(t, isError, text)
			assert.Contains(t, text, "queryid: "+queryID)
			assert.Contains(t, text, "engine: "+engine)
			assert.Contains(t, text, "service: "+serviceName)
			assert.Contains(t, text, "fingerprint:")
			assert.Contains(t, text, "filter_by="+queryID)
		})
	}

	t.Run("InvalidWindow", func(t *testing.T) {
		t.Parallel()

		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"period_from": "yesterday"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input"), text)
	})
}

func TestExplainMySQLRunsLive(t *testing.T) {
	t.Parallel()

	session := connect(t)

	// MySQL has no stored plans, so the tool runs the live mysql_explain_json
	// action without looking for one. pmm-agent explains DML only, so take the
	// busiest DML row: the top row by load can be another statement.
	serviceID, serviceName, _ := topQueryID(t, session, "mysql")
	text, isError := callText(t, session, "pmm_top_queries", map[string]any{"service_name": serviceName, "period_from": "now-12h", "limit": 100})
	require.False(t, isError, text)
	m := regexp.MustCompile(`(?m)^\d+\. \[([0-9A-Za-z]+)\].*\n {3}(?i:SELECT|INSERT|UPDATE|DELETE|REPLACE)\b`).FindStringSubmatch(text)
	if m == nil {
		t.Skipf("no DML query in QAN for %s yet", serviceName)
	}
	queryID := m[1]

	text, isError = callText(t, session, "pmm_get_explain", map[string]any{"service_id": serviceID, "queryid": queryID})
	assert.NotContains(t, text, "Stored plan")
	if isError {
		// Without a stored example the fingerprint's placeholders make MySQL
		// answer 1064, which is mapped to invalid_input with remediation text.
		assert.True(t, strings.HasPrefix(text, "error: invalid_input") || strings.HasPrefix(text, "error: no_query_source"), text)
	} else {
		assert.Contains(t, text, "EXPLAIN (json, mysql")
	}
}

// serviceListQuery is the query behind pmm_inventory's list, for the last hour.
const serviceListQuery = "max by (service_id, service_name, service_type, node_id, node_name, environment, cluster) (tlast_over_time(up[3600s]))"

// adminAuthorization returns the Basic Authorization value of the user in the test server URL.
func adminAuthorization() string {
	if pmmapitests.BaseURL.User == nil {
		return ""
	}
	password, _ := pmmapitests.BaseURL.User.Password()
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(pmmapitests.BaseURL.User.Username()+":"+password))
}

// getJSON GETs a path of the server under test with authorization and decodes the JSON answer into out.
func getJSON(t *testing.T, authorization, path string, query url.Values, out any) {
	t.Helper()

	u := pmmapitests.BaseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	u.User = nil
	req, err := http.NewRequestWithContext(pmmapitests.Context, http.MethodGet, u.String(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", authorization)
	res, err := httpClient(false).Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode, "GET %s: %s", path, body)
	require.NoError(t, json.Unmarshal(body, out))
}

// upRows returns the rows of the service-list query for the last hour, read through the Metrics datasource with authorization.
func upRows(t *testing.T, authorization string) []map[string]string {
	t.Helper()

	var ds struct {
		UID string `json:"uid"`
	}
	getJSON(t, authorization, "graph/api/datasources/name/Metrics", nil, &ds)
	require.NotEmpty(t, ds.UID)

	var answer struct {
		Data struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
			} `json:"result"`
		} `json:"data"`
	}
	getJSON(t, authorization, "graph/api/datasources/proxy/uid/"+ds.UID+"/api/v1/query", url.Values{"query": {serviceListQuery}}, &answer)

	rows := make([]map[string]string, 0, len(answer.Data.Result))
	for _, r := range answer.Data.Result {
		rows = append(rows, r.Metric)
	}
	return rows
}

func TestInventory(t *testing.T) {
	t.Parallel()

	rows := upRows(t, adminAuthorization())
	require.NotEmpty(t, rows)

	var got struct {
		Page struct {
			Total int `json:"total"`
		} `json:"page"`
	}
	text := callResult(t, connect(t), "pmm_inventory", map[string]any{"limit": 200}, &got)

	ids := map[string]bool{}
	for _, row := range rows {
		id := row["service_id"]
		if id == "" {
			continue
		}
		ids[id] = true
		assert.Contains(t, text, "id="+id)
		assert.Contains(t, text, "- "+row["service_name"]+" | "+row["service_type"]+" ")
	}
	assert.Len(t, ids, got.Page.Total)

	assert.NotContains(t, text, "address=")
	lastSeen := regexp.MustCompile(`last_seen=(\S+)`).FindAllStringSubmatch(text, -1)
	require.Len(t, lastSeen, len(ids))
	for _, m := range lastSeen {
		seen, err := time.Parse(time.RFC3339, m[1])
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), seen, time.Hour)
	}

	t.Run("EngineFilter", func(t *testing.T) {
		t.Parallel()

		text, isError := callText(t, connect(t), "pmm_inventory", map[string]any{"engine": "mysql", "limit": 200})
		assert.False(t, isError)
		assert.NotContains(t, text, "| postgresql ")
		assert.NotContains(t, text, "| mongodb ")
		assert.NotContains(t, text, "| proxysql ")
	})

	t.Run("InvalidEngine", func(t *testing.T) {
		t.Parallel()

		text, isError := callText(t, connect(t), "pmm_inventory", map[string]any{"engine": "oracle"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input"), text)
	})
}

func TestTopQueriesGroupBy(t *testing.T) {
	t.Parallel()

	session := connect(t)
	_, serviceName, _ := topQueryID(t, session, "mysql")
	args := map[string]any{"service_name": serviceName, "period_from": "now-12h", "group_by": "username", "columns": []string{"rows_examined"}, "limit": 1}
	text, isError := callText(t, session, "pmm_top_queries", args)
	require.False(t, isError, text)
	assert.Regexp(t, `^Rows 1-1 of \d+ \(group_by=username, order_by=load\):`, text)
	assert.Contains(t, text, "rows_examined_avg=")
	assert.NotContains(t, text, "\n   ", "no fingerprint line unless group_by=queryid")

	args["offset"] = 1
	text, isError = callText(t, session, "pmm_top_queries", args)
	require.False(t, isError, text)
	assert.Regexp(t, `^(Rows 2-2 of \d+|No rows at offset 1; the report has 1\.)`, text)
}

// TestSchemaFromQueryID fetches a known join's tables from its query_ref, which pmm_get_schema finds in a live EXPLAIN plan.
func TestSchemaFromQueryID(t *testing.T) {
	if !pmmapitests.RunMCPWorkloadTests {
		t.Skip("Needs a known workload whose join has a query example in QAN: load managed/services/mcp/testdata/explain/schema.sql " +
			"into a MySQL 8.0 service monitored with perfschema and query examples on, then run join-aliases.sql from the same " +
			"folder 100 times in one mysql session (database shop) that ends with SELECT SLEEP(10). pmm-agent reads the example " +
			"from performance_schema.events_statements_history (consumer events_statements_history, on by default) every 5 " +
			"seconds, and that table drops a session's statements when it closes. Wait 2 minutes for QAN, then set " +
			"-pmm.run-mcp-workload-tests [PMM_RUN_MCP_WORKLOAD_TESTS].")
	}
	t.Parallel()

	session := connect(t)
	res, err := inventoryClient.Default.ServicesService.ListServices(&services_service.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	// The join of join-aliases.sql; schema.sql names both tables too, but joins nothing.
	join := regexp.MustCompile(`(?is)\borders\b.*\bjoin\b.*\bcustomers\b`)
	var ref map[string]any
	for _, svc := range res.Payload.Mysql {
		var top struct {
			Rows []struct {
				Fingerprint string         `json:"fingerprint"`
				QueryRef    map[string]any `json:"query_ref"`
			} `json:"rows"`
		}
		args := map[string]any{"service_name": svc.ServiceName, "search": "customers", "period_from": "now-12h", "limit": 100}
		callResult(t, session, "pmm_top_queries", args, &top)
		for _, row := range top.Rows {
			if row.QueryRef != nil && join.MatchString(row.Fingerprint) {
				ref = row.QueryRef
				break
			}
		}
		if ref != nil {
			break
		}
	}
	require.NotNil(t, ref, "no pmm_top_queries row joins orders and customers: load the workload of testdata/explain first")

	var schema struct {
		Tables []struct {
			Database   string `json:"database"`
			Name       string `json:"name"`
			Definition string `json:"definition"`
		} `json:"tables"`
		Unresolved []string `json:"unresolved"`
		Note       string   `json:"note"`
	}
	called, text := callTool(t, session, "pmm_get_schema", map[string]any{"query_ref": ref})
	if called.IsError && strings.HasPrefix(text, "error: invalid_input\n") && strings.Contains(text, "placeholders") {
		require.FailNow(t, "no query example stored for the join; keep the workload's mysql session open past pmm-agent's "+
			"5-second read of events_statements_history, with query examples on, and wait for QAN, as the skip message says", text)
	}
	require.False(t, called.IsError, text)
	b, err := json.Marshal(called.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &schema))
	names := make([]string, 0, len(schema.Tables))
	for _, tbl := range schema.Tables {
		names = append(names, tbl.Database+"."+tbl.Name)
		assert.Contains(t, tbl.Definition, "CREATE TABLE", tbl.Name)
	}
	assert.ElementsMatch(t, []string{"shop.orders", "shop.customers"}, names)
	assert.Equal(t, 1, strings.Count(text, "### shop.orders\n"))
	assert.Equal(t, 1, strings.Count(text, "### shop.customers\n"))
	assert.Empty(t, schema.Unresolved)
	assert.NotEmpty(t, schema.Note, "tables from a live plan are labelled")
}

// maxMetricsBytes is the budget of one metrics response: its text and structured content together.
const maxMetricsBytes = 16 << 10

// monitoredService returns the id and name of a registered service of the engine, skipping the test when there is none.
func monitoredService(t *testing.T, engine string) (string, string) {
	t.Helper()

	res, err := inventoryClient.Default.ServicesService.ListServices(&services_service.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	switch engine {
	case "mysql":
		for _, svc := range res.Payload.Mysql {
			return svc.ServiceID, svc.ServiceName
		}
	case "postgresql":
		for _, svc := range res.Payload.Postgresql {
			if svc.ServiceName == "pmm-server-postgresql" {
				continue
			}
			return svc.ServiceID, svc.ServiceName
		}
	case "mongodb":
		for _, svc := range res.Payload.Mongodb {
			return svc.ServiceID, svc.ServiceName
		}
	}
	t.Skipf("no %s service registered", engine)
	return "", ""
}

func TestServiceHealth(t *testing.T) {
	session := connect(t)
	for _, engine := range []string{"mysql", "postgresql", "mongodb"} {
		t.Run(engine, func(t *testing.T) {
			serviceID, serviceName := monitoredService(t, engine)

			var health struct {
				ServiceID   string `json:"service_id"`
				ServiceName string `json:"service_name"`
				Engine      string `json:"engine"`
				Signals     []struct {
					Name   string `json:"name"`
					LastAt string `json:"last_at"`
					NoData bool   `json:"no_data"`
					Error  any    `json:"error"`
				} `json:"signals"`
			}
			args := map[string]any{"service_id": serviceID, "period_from": "now-6h"}
			callResult(t, session, "pmm_service_health", args, &health)

			assert.Equal(t, serviceID, health.ServiceID)
			assert.Equal(t, serviceName, health.ServiceName)
			assert.Equal(t, engine, health.Engine)
			require.NotEmpty(t, health.Signals)
			withData := 0
			for _, sig := range health.Signals {
				assert.Nil(t, sig.Error, "signal %s", sig.Name)
				assert.True(t, sig.LastAt != "" || sig.NoData, "signal %s has neither last_at nor no_data", sig.Name)
				if sig.LastAt != "" {
					withData++
				}
			}
			assert.Positive(t, withData, "no signal has data")
		})
	}
}

func TestQueryMetricsBudget(t *testing.T) {
	_, serviceName := monitoredService(t, "mysql")
	session := connect(t)

	// 3 minutes at the default 1 s step give every series many points on a new server and stay well under the 4 MiB read on an old one.
	query := `{__name__=~"mysql_global_status_.*", service_name="` + serviceName + `"}`
	res, text := callTool(t, session, "pmm_query_metrics", map[string]any{"query": query, "period_from": "now-3m"})
	require.False(t, res.IsError, text)

	structured, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out struct {
		Summarized  bool `json:"summarized"`
		TotalSeries int  `json:"total_series"`
	}
	require.NoError(t, json.Unmarshal(structured, &out))

	if out.TotalSeries == 0 {
		t.Skipf("no mysql_global_status_ series for %s yet", serviceName)
	}
	assert.True(t, out.Summarized, "3 minutes of many series at a 1 s step must come back as summaries")
	assert.Greater(t, out.TotalSeries, 1)
	assert.LessOrEqual(t, len(text)+len(structured), maxMetricsBytes)
}

func TestListMetricsSearch(t *testing.T) {
	session := connect(t)
	type listed struct {
		Names []struct {
			Name string `json:"name"`
		} `json:"names"`
		Groups []struct {
			Prefix string `json:"prefix"`
		} `json:"groups"`
		Page struct {
			Total int `json:"total"`
		} `json:"page"`
	}

	t.Run("Substring", func(t *testing.T) {
		// The MongoDB exporter has about 175 names with this text, few enough to come back as names, not groups.
		const search = "mongodb_ss_wt_cache"
		var out listed
		callResult(t, session, "pmm_list_metrics", map[string]any{"search": search, "limit": 1000}, &out)
		require.Empty(t, out.Groups, "names came back as groups, so the substring filter is not tested")
		if len(out.Names) == 0 {
			t.Skip("no MongoDB metric names")
		}
		for _, n := range out.Names {
			assert.Contains(t, n.Name, search)
		}
	})

	t.Run("RegexIsLiteral", func(t *testing.T) {
		var out listed
		callResult(t, session, "pmm_list_metrics", map[string]any{"search": ".*"}, &out)
		assert.Empty(t, out.Names)
		assert.Zero(t, out.Page.Total)
	})

	// Counts raw values, since names beyond limit come back as groups.
	t.Run("LimitHonoured", func(t *testing.T) {
		const limit = 5
		var out struct {
			Values []string `json:"values"`
			Page   struct {
				Total     int  `json:"total"`
				Limit     int  `json:"limit"`
				Truncated bool `json:"truncated"`
			} `json:"page"`
		}
		callResult(t, session, "pmm_list_metrics", map[string]any{"kind": "values", "label": "__name__", "limit": limit}, &out)
		if out.Page.Total <= limit {
			t.Skipf("only %d metric names", out.Page.Total)
		}
		assert.LessOrEqual(t, len(out.Values), limit)
		assert.Equal(t, limit, out.Page.Limit)
		assert.True(t, out.Page.Truncated)
		assert.LessOrEqual(t, out.Page.Total, 1001, "VictoriaMetrics must honour the limit the tool sends")
	})
}
